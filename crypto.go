package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/chacha20"
)

const zeroKID = "00000000000000000000000000000000"

var shakaMissingKeyIDRE = regexp.MustCompile(`(?i)Key for key_id=([0-9a-f]{32}) was not found`)

func decryptSegment(data []byte, enc EncryptInfo) ([]byte, error) {
	switch enc.Method {
	case "", EncryptNone:
		return data, nil
	case EncryptAES128:
		return decryptAESCBC(data, enc.Key, enc.IV)
	case EncryptAES128ECB:
		return decryptAESECB(data, enc.Key)
	case EncryptChaCha20:
		return decryptChaCha20Per1024(data, enc.Key, enc.IV)
	case EncryptCENC, EncryptSampleAES, EncryptSampleCTR, EncryptUnknown:
		// long: 上游遇到无法识别的 HLS 加密时不会尝试伪解密，而是保留分片并强制二进制合并，让用户后续可用外部工具处理。
		return data, nil
	default:
		// long: C# 枚举可解析出未定义数字 METHOD，原版下载器 switch 不匹配时会保留原始分片；这里不能把这类边界值升级成下载失败。
		return data, nil
	}
}

func decryptAESCBC(data, key, iv []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("AES-128 key 长度应为 16")
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("AES CBC IV 长度应为 16")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度不是 AES block size 的整数倍")
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	return pkcs7Unpad(out, aes.BlockSize)
}

func decryptAESECB(data, key []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("AES-128 key 长度应为 16")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度不是 AES block size 的整数倍")
	}
	out := make([]byte, len(data))
	for bs, be := 0, aes.BlockSize; bs < len(data); bs, be = bs+aes.BlockSize, be+aes.BlockSize {
		block.Decrypt(out[bs:be], data[bs:be])
	}
	return pkcs7Unpad(out, aes.BlockSize)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("PKCS7 数据长度无效")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("PKCS7 padding 无效")
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("PKCS7 padding 不一致")
		}
	}
	return data[:len(data)-pad], nil
}

func decryptChaCha20Per1024(data, key, nonce []byte) ([]byte, error) {
	if len(key) != chacha20.KeySize {
		return nil, fmt.Errorf("CHACHA20 key 长度应为 32")
	}
	if len(nonce) == 8 {
		nonce = append([]byte{0, 0, 0, 0}, nonce...)
	}
	if len(nonce) != chacha20.NonceSize {
		return nil, fmt.Errorf("CHACHA20 nonce 长度应为 12 或 8")
	}
	out := make([]byte, len(data))
	for start := 0; start < len(data); start += 1024 {
		end := start + 1024
		if end > len(data) {
			end = len(data)
		}
		c, err := chacha20.NewUnauthenticatedCipher(key, nonce)
		if err != nil {
			return nil, err
		}
		// long: 上游每 1024 字节重新创建 ChaCha20 流，兼容这种非标准 HLS 分片加密方式时必须保留同样的重置节奏。
		c.XORKeyStream(out[start:end], data[start:end])
	}
	return out, nil
}

func hasExternalMP4Encryption(s StreamSpec) bool {
	if s.Playlist == nil {
		return false
	}
	for _, part := range s.Playlist.Parts {
		for _, seg := range part.Segments {
			switch seg.Encrypt.Method {
			case EncryptCENC, EncryptSampleAES, EncryptSampleCTR:
				return true
			}
		}
	}
	if s.Playlist.MediaInit != nil {
		switch s.Playlist.MediaInit.Encrypt.Method {
		case EncryptCENC, EncryptSampleAES, EncryptSampleCTR:
			return true
		}
	}
	return false
}

func hasUnknownEncryption(streams []StreamSpec) bool {
	for _, s := range streams {
		if s.Playlist == nil {
			continue
		}
		if s.Playlist.MediaInit != nil && s.Playlist.MediaInit.Encrypt.Method == EncryptUnknown {
			return true
		}
		for _, part := range s.Playlist.Parts {
			for _, seg := range part.Segments {
				if seg.Encrypt.Method == EncryptUnknown {
					return true
				}
			}
		}
	}
	return false
}

func hasCENCEncryption(streams []StreamSpec) bool {
	for _, s := range streams {
		if s.Playlist == nil {
			continue
		}
		if s.Playlist.MediaInit != nil && s.Playlist.MediaInit.Encrypt.Method == EncryptCENC {
			return true
		}
		for _, part := range s.Playlist.Parts {
			for _, seg := range part.Segments {
				if seg.Encrypt.Method == EncryptCENC {
					return true
				}
			}
		}
	}
	return false
}

func hasFMP4Media(streams []StreamSpec) bool {
	for _, s := range streams {
		if s.Playlist == nil || s.Playlist.MediaInit == nil {
			continue
		}
		if s.MediaType != nil && *s.MediaType == MediaSubtitles {
			continue
		}
		return true
	}
	return false
}

func decryptMP4Output(path string, opt Options) (string, error) {
	return decryptMP4File(path, opt, "", "")
}

func decryptMP4File(path string, opt Options, kid string, initPath string) (string, error) {
	multiDRM := false
	if initPath != "" {
		detectedKID, detectedMultiDRM := extractDefaultKIDInfoFromFile(initPath)
		if kid == "" {
			kid = detectedKID
		}
		multiDRM = detectedMultiDRM
	}
	if kid == "" {
		detectedKID, detectedMultiDRM := extractDefaultKIDInfoFromFile(path)
		kid = detectedKID
		multiDRM = detectedMultiDRM
	} else if !multiDRM {
		_, detectedMultiDRM := extractDefaultKIDInfoFromFile(path)
		multiDRM = detectedMultiDRM
	}
	engine := strings.ToUpper(opt.DecryptionEngine)
	if strings.HasSuffix(path, "_init.mp4") && engine != "MP4DECRYPT" && engine != "" {
		// long: shaka-packager/ffmpeg 需要 init+media 才能解密，原版遇到单独 _init.mp4 会直接跳过，避免把 init 文件误交给外部工具。
		return path, nil
	}
	bin := opt.DecryptionBinaryPath
	if bin == "" {
		switch engine {
		case "SHAKA_PACKAGER":
			bin = firstExecutable("shaka-packager", "packager-linux-x64", "packager-osx-x64", "packager-win-x64")
		case "FFMPEG":
			bin = opt.FFmpegBinaryPath
			if bin == "" {
				bin = "ffmpeg"
			}
		default:
			bin = firstExecutable("mp4decrypt")
		}
	}
	if bin == "" {
		return path, fmt.Errorf("%s", decryptToolNotFoundText(opt, engine))
	}
	if kid == "" && engine == "SHAKA_PACKAGER" {
		if detected, err := detectKIDWithShaka(path, bin); err == nil && detected != "" {
			kid = detected
			multiDRM = false
		}
	}
	keys := collectDecryptKeys(opt, kid)
	if len(keys) == 0 {
		return path, nil
	}
	dest := strings.TrimSuffix(path, filepath.Ext(path)) + "_dec" + filepath.Ext(path)
	var cmd *exec.Cmd
	inputPath := path
	tmpFile := ""
	tmpEncFile := ""
	tmpDecFile := ""
	mp4decryptTmp := false
	selectedKey, ok := selectDecryptKeyPair(keys, kid)
	if !ok {
		return path, nil
	}
	switch engine {
	case "SHAKA_PACKAGER":
		label, keyID, key := normalizeKeyForShakaWithFlags(selectedKey, kid, multiDRM)
		if initPath != "" {
			tmpFile = strings.TrimSuffix(path, filepath.Ext(path)) + ".itmp" + filepath.Ext(path)
			if err := binaryMerge([]string{initPath, path}, tmpFile); err != nil {
				return path, err
			}
			inputPath = tmpFile
		}
		keyArg := fmt.Sprintf("key_id=%s:key=%s", keyID, key)
		if label != "" {
			keyArg = fmt.Sprintf("label=%s:%s", label, keyArg)
		}
		cmd = exec.Command(bin, "--quiet", "--enable_raw_key_decryption",
			fmt.Sprintf("input=%s,stream=0,output=%s", inputPath, dest),
			"--keys", keyArg)
	case "FFMPEG":
		_, key := normalizeKeyForExternal(selectedKey, false, kid)
		if initPath != "" {
			tmpFile = strings.TrimSuffix(path, filepath.Ext(path)) + ".itmp" + filepath.Ext(path)
			if err := binaryMerge([]string{initPath, path}, tmpFile); err != nil {
				return path, err
			}
			inputPath = tmpFile
		}
		cmd = exec.Command(bin, "-loglevel", "error", "-nostdin", "-decryption_key", key, "-i", inputPath, "-c", "copy", dest)
	default:
		args := []string{}
		for _, k := range keys {
			args = append(args, "--key", normalizeMP4DecryptKeyWithFlags(k, kid, multiDRM))
		}
		workDir := filepath.Dir(path)
		tmpPath, err := tempPathWithExt(workDir, filepath.Ext(path))
		if err != nil {
			return path, err
		}
		tmpEncFile = tmpPath
		tmpDecFile = strings.TrimSuffix(tmpEncFile, filepath.Ext(tmpEncFile)) + "_dec" + filepath.Ext(tmpEncFile)
		if err := os.Rename(path, tmpEncFile); err != nil {
			return path, err
		}
		mp4decryptTmp = true
		if initPath != "" {
			infoPath := initPath
			// long: mp4decrypt 在非 ASCII 路径下更容易失败；上游会切到媒体目录并把同目录 init 改为相对文件名，这里保持同样的调用形态。
			if sameDirectory(filepath.Dir(initPath), workDir) {
				infoPath = filepath.Base(initPath)
			}
			args = append(args, "--fragments-info", infoPath)
		}
		args = append(args, filepath.Base(tmpEncFile), filepath.Base(tmpDecFile))
		cmd = exec.Command(bin, args...)
		cmd.Dir = workDir
	}
	out, err := cmd.CombinedOutput()
	if tmpFile != "" {
		_ = os.Remove(tmpFile)
	}
	if mp4decryptTmp {
		if _, statErr := os.Stat(tmpEncFile); statErr == nil {
			_ = os.Rename(tmpEncFile, path)
		}
		if err == nil {
			if _, statErr := os.Stat(tmpDecFile); statErr == nil {
				_ = os.Remove(dest)
				if moveErr := os.Rename(tmpDecFile, dest); moveErr != nil {
					return path, moveErr
				}
			}
		} else {
			_ = os.Remove(tmpDecFile)
		}
	}
	if err != nil {
		return path, fmt.Errorf("%s: %v\n%s", tr(opt, "decryptionFailed"), err, string(out))
	}
	if err := replaceFile(dest, path); err != nil {
		return path, err
	}
	return path, nil
}

func detectKIDWithShaka(path string, bin string) (string, error) {
	if path == "" || bin == "" {
		return "", nil
	}
	tmp := path + ".tmp.webm"
	// long: 部分 fMP4/WebM init 读不到 tenc/PSSH KID；上游会用 shaka 的失败信息反查 key_id，以便继续匹配用户 key-file。
	cmd := exec.Command(bin,
		"--quiet",
		"--enable_raw_key_decryption",
		fmt.Sprintf("input=%s,stream=0,output=%s", path, tmp),
		"--keys", fmt.Sprintf("key_id=%s:key=%s", zeroKID, zeroKID),
	)
	out, err := cmd.CombinedOutput()
	_ = os.Remove(tmp)
	match := shakaMissingKeyIDRE.FindStringSubmatch(string(out))
	if len(match) >= 2 {
		return strings.ToLower(match[1]), nil
	}
	if err != nil {
		return "", err
	}
	return "", nil
}

func tempPathWithExt(dir, ext string) (string, error) {
	f, err := os.CreateTemp(dir, "n_m3u8dl_*"+ext)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
}

func sameDirectory(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return filepath.Clean(absA) == filepath.Clean(absB)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if removeErr := os.Remove(dst); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(src, dst)
}

func collectDecryptKeys(opt Options, kid string) []string {
	var keys []string
	keys = append(keys, opt.Keys...)
	if opt.KeyTextFile != "" && kid != "" {
		if b, err := os.ReadFile(opt.KeyTextFile); err == nil {
			fmt.Println(tr(opt, "searchKey"))
			for _, line := range strings.Split(string(b), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				// long: 上游 SearchKeyFromFileAsync 只在已知 KID 时查找 key 文件，并按 StartsWith 精确匹配，避免无 KID 分片误用 key 文件里的任意密钥。
				if strings.HasPrefix(line, kid) {
					keys = append(keys, line)
					fmt.Printf("OK %s\n", line)
					break
				}
			}
		}
	}
	return keys
}

func decryptToolNotFoundText(opt Options, engine string) string {
	switch strings.ToUpper(engine) {
	case "SHAKA_PACKAGER":
		return tr(opt, "shakaPackagerNotFound")
	case "FFMPEG":
		return tr(opt, "ffmpegNotFound")
	default:
		return tr(opt, "mp4decryptNotFound")
	}
}

func selectDecryptKeyPair(keys []string, kid string) (string, bool) {
	if len(keys) == 0 {
		return "", false
	}
	if kid != "" {
		for _, key := range keys {
			// long: 上游运行时用 StartsWith(kid) 精确匹配当前 KID；这里保持大小写敏感，避免把 key-file 或外部注入的非归一化 KID 当作同一把密钥。
			if strings.HasPrefix(strings.TrimSpace(key), kid) {
				return strings.TrimSpace(key), true
			}
		}
	}
	if strings.EqualFold(kid, zeroKID) {
		return strings.TrimSpace(keys[0]), true
	}
	if len(keys) == 1 && !strings.Contains(keys[0], ":") {
		if kid != "" {
			return kid + ":" + strings.TrimSpace(keys[0]), true
		}
		return strings.TrimSpace(keys[0]), true
	}
	return "", false
}

func normalizeMP4DecryptKey(input string, detectedKid string) string {
	return normalizeMP4DecryptKeyWithFlags(input, detectedKid, false)
}

func normalizeMP4DecryptKeyWithFlags(input string, detectedKid string, multiDRM bool) string {
	input = strings.TrimSpace(input)
	key := input
	if left, right, ok := strings.Cut(input, ":"); ok {
		if multiDRM {
			return "1:" + right
		}
		if strings.EqualFold(detectedKid, zeroKID) {
			return "1:" + right
		}
		if left != "" {
			return input
		}
		key = right
	}
	if strings.EqualFold(detectedKid, zeroKID) {
		return "1:" + key
	}
	if multiDRM {
		return "1:" + key
	}
	if strings.Contains(input, ":") {
		return input
	}
	if detectedKid != "" {
		return detectedKid + ":" + input
	}
	return "1:" + input
}

func normalizeKeyForExternal(input string, needKid bool, detectedKid string) (string, string) {
	input = strings.TrimSpace(input)
	if kid, key, ok := strings.Cut(input, ":"); ok {
		return kid, key
	}
	if needKid {
		if detectedKid != "" {
			return detectedKid, input
		}
		return "00000000000000000000000000000000", input
	}
	return "", input
}

func normalizeKeyForShaka(input string, detectedKid string) (label string, keyID string, key string) {
	return normalizeKeyForShakaWithFlags(input, detectedKid, false)
}

func normalizeKeyForShakaWithFlags(input string, detectedKid string, multiDRM bool) (label string, keyID string, key string) {
	input = strings.TrimSpace(input)
	keyID = detectedKid
	key = input
	if kid, rawKey, ok := strings.Cut(input, ":"); ok {
		keyID = kid
		key = rawKey
	}
	if strings.EqualFold(detectedKid, zeroKID) {
		return "1", zeroKID, key
	}
	if multiDRM {
		return "1", zeroKID, key
	}
	if keyID == "" {
		keyID = zeroKID
	}
	return "", keyID, key
}

func firstExecutable(names ...string) string {
	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
