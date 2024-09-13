package models

type User struct {
	Id       uint64 `json:"id" validate:"required" gorm:"primarykey" example:"1"`
	Username string `json:"username" validate:"required" example:"ex4ample"`
	Mail     string `json:"-"`
	/*
			HasImage     bool      `json:"hasImage" validate:"required" example:"true"`
		PasswordHash string    `json:"-"`
		IsVerified   bool      `json:"-"`
		CreatedUtc   time.Time `json:"-"`
		// To verify account by mail
		VerificationCode string `json:"-"`
	*/
	// RSA public key
	PublicKey string `json:"publicKey" validate:"required" example:"RSA_PUBLIC_KEY_PEM"`
	/*
		// To recover account when password is lost, encrypted (AES) by PBKDF2 derivation from account password
		EncryptedMasterKey string `json:"-"`
		MasterKeyHash      string `json:"-"`
		// RSA private key encrypted (AES) by master key
		EncryptedPrivateKey string `json:"-"`
	*/
	Admin bool `json:"admin" validate:"required" example:"false"`
}

// Return new user.
//
//	@param username
//	@param mail
//	@param publicKey
//	@param admin
//	@return *User
func NewUser(username string, mail string, publicKey string, admin bool) *User {
	u := new(User)
	u.Username = username
	u.Mail = mail
	u.PublicKey = publicKey
	u.Admin = admin
	return u
	/*
		u.HasImage = false

			// Password hash
			passwordBcryptBytes, err := bcrypt.GenerateFromPassword([]byte(register.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			u.PasswordHash = base64.StdEncoding.EncodeToString(passwordBcryptBytes)

			u.IsVerified = false
			u.VerificationCode = utils.RandomString(32)
	*/

	/*
		// Keys
		// RSA: https://www.systutorials.com/how-to-generate-rsa-private-and-public-key-pair-in-go-lang/
		// Generate private key
		privatekey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, err
		}
		// Public key
		publickey := &privatekey.PublicKey
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publickey)
		if err != nil {
			return nil, err
		}
		publicKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}
		u.PublicKey = string(pem.EncodeToMemory(publicKeyBlock))
			// Master key
			masterKey := utils.RandomString(32)
			masterKeyIv := md5.Sum([]byte(register.Password))
			masterKeyKey := pbkdf2.Key([]byte(register.Password), masterKeyIv[:], 310000, 32, sha256.New)
			encryptedMasterKey, err := utils.Aes256Encrypt([]byte(masterKey), masterKeyKey, masterKeyIv[:])
			if err != nil {
				return nil, err
			}
			u.EncryptedMasterKey = base64.StdEncoding.EncodeToString(encryptedMasterKey)
			// Master key hash
			masterKeyBcryptBytes, err := bcrypt.GenerateFromPassword([]byte(masterKey), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			u.MasterKeyHash = base64.StdEncoding.EncodeToString(masterKeyBcryptBytes)
			// Private key
			var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(privatekey)
			privateKeyBlock := &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: privateKeyBytes,
			}
			privateKeyIv := md5.Sum([]byte(masterKey))
			encryptedPrivateKey, err := utils.Aes256Encrypt(pem.EncodeToMemory(privateKeyBlock), []byte(masterKey), privateKeyIv[:])
			if err != nil {
				return nil, err
			}
			u.EncryptedPrivateKey = base64.StdEncoding.EncodeToString(encryptedPrivateKey)

			u.CreatedUtc = time.Now().UTC()
	*/
}
