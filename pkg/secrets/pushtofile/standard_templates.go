package pushtofile

import "fmt"

type standardTemplate struct {
	template      string
	fileFormat    string
	validateAlias func(alias string) error
}

func (s standardTemplate) ValidateAlias(alias string) error {
	if s.validateAlias == nil {
		return nil
	}

	return s.validateAlias(alias)
}

var standardTemplates = map[string]standardTemplate{
	"yaml":   {template: yamlTemplate, validateAlias: validateYAMLKey},
	"json":   {template: jsonTemplate, validateAlias: validateJSONKey},
	"dotenv": {template: dotenvTemplate, validateAlias: validateBashVarName},
	"bash":   {template: bashTemplate, validateAlias: validateBashVarName},
}

// FileTemplateForFormat returns the template for a file format, after ensuring the
// standard template exists and validating secret spec aliases against it
func FileTemplateForFormat(
	fileFormat string,
	secretSpecs []SecretSpec,
) (string, error) {
	stdTemplate, ok := standardTemplates[fileFormat]
	if !ok {
		return "", fmt.Errorf(`unrecognized standard file format, "%s"`, fileFormat)
	}

	for _, s := range secretSpecs {
		err := stdTemplate.ValidateAlias(s.Alias)
		if err != nil {
			return "", err
		}
	}

	return stdTemplate.template, nil
}
