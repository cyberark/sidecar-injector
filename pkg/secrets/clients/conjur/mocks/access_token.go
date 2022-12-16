package mocks

// Mocks a Conjur access token.
type MockAccessToken struct{}

// Returns an arbitrary byte array as an access token data as we don't really need it
func (accessToken MockAccessToken) Read() ([]byte, error) {
	return []byte("someAccessToken"), nil
}

// This method implementation is only so MockAccessToken will implement the MockAccessToken interface
func (accessToken MockAccessToken) Write(Data []byte) error {
	return nil
}

// This method implementation is only so MockAccessToken will implement the MockAccessToken interface
func (accessToken MockAccessToken) Delete() error {
	return nil
}
