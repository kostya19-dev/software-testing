package lab5_test

func (s *world) requestRecovery() error {
	var result struct{ Challenge string }
	if err := s.request("POST", "/auth/recover", map[string]string{"email": s.email}, 202, &result, false); err != nil {
		return err
	}
	s.recovery = result.Challenge
	return nil
}

func (s *world) reset() error {
	code, err := s.mailCode(s.recovery)
	if err != nil {
		return err
	}
	s.recoveryCode = code
	if err = s.request("POST", "/auth/reset", map[string]string{"email": s.email, "challenge": s.recovery, "code": code, "new_password": s.replacement}, 200, nil, false); err != nil {
		return err
	}
	s.password = s.replacement
	return nil
}

func (s *world) changeWith(old, new string, status int) error {
	return s.request("POST", "/auth/password", map[string]string{"password": old, "new_password": new}, status, nil, true)
}
