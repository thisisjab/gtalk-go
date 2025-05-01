package validator

type Validator struct {
	errors map[string]string
}

func New() *Validator {
	return &Validator{
		errors: make(map[string]string),
	}
}

func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

func (v *Validator) AddError(key, message string) {
	if _, exists := v.errors[key]; !exists {
		v.errors[key] = message
	}
}

func (v *Validator) Errors() map[string]string {
	return v.errors
}

func (v *Validator) Check(condition bool, key, msg string) {
	if !condition {
		v.AddError(key, msg)
	}
}
