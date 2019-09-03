package collection

// An ActivationRenderer customizes the source content from the repository before it is applied
type ActivationRenderer interface {
	//Render processes the yaml source content before it is unmarshaled into an object model
	Render(b []byte, context map[string]interface{}) ([]byte, error)
}
