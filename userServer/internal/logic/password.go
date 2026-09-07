package logic

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码不能少于 8 位")
	}

	if len(password) > 64 {
		return errors.New("密码不能超过 64 位")
	}

	for i := 0; i < len(password); i++ {
		c := password[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return errors.New("密码只能包含大小写的英文字母和数字")
		}
	}
	return nil
}

func generatePasswordDigest(password string) (string, error) {
	if password == "" {
		return "", errors.New("密码不能为空")
	}
	if len([]byte(password)) > 72 {
		return "", errors.New("密码长度超过限制")
	}
	if e := validatePassword(password); e != nil {
		return "", e
	}
	digest, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}
	return string(digest), nil
}

func comparePassword(digest, password string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(digest),
		[]byte(password),
	)

	return err == nil
}
