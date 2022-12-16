package pushtofile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

const secretGroupPrefix = "conjur.org/conjur-secrets."
const secretGroupPolicyPathPrefix = "conjur.org/conjur-secrets-policy-path."
const secretGroupFileTemplatePrefix = "conjur.org/secret-file-template."
const secretGroupFilePathPrefix = "conjur.org/secret-file-path."
const secretGroupFileFormatPrefix = "conjur.org/secret-file-format."
const secretGroupFilePermissionsPrefix = "conjur.org/secret-file-permissions."

const defaultFilePermissions os.FileMode = 0644
const maxFilenameLen = 255

// Config is used during SecretGroup creation, and contains default values for
// secret file and template file base paths, along with mockable functions for
// reading template files.
type Config struct {
	secretsBasePath   string
	templatesBasePath string
	openReadCloser    openReadCloserFunc
	pullFromReader    pullFromReaderFunc
}

// SecretGroup incorporates all of the information about a secret group
// that has been parsed from that secret group's Annotations.
type SecretGroup struct {
	Name             string
	FilePath         string
	FileTemplate     string
	FileFormat       string
	PolicyPathPrefix string
	FilePermissions  os.FileMode
	SecretSpecs      []SecretSpec
}

// ResolvedSecretSpecs resolves all of the secret paths for a secret
// group by prepending each path with that group's policy path prefix.
// Updates the Path of each SecretSpec in the field SecretSpecs.
func resolvedSecretSpecs(policyPathPrefix string, secretSpecs []SecretSpec) []SecretSpec {
	if len(policyPathPrefix) != 0 {
		for i := range secretSpecs {
			secretSpecs[i].Path = strings.TrimSuffix(policyPathPrefix, "/") +
				"/" + strings.TrimPrefix(secretSpecs[i].Path, "/")
		}
	}

	return secretSpecs
}

// PushToFile uses the configuration on a secret group to inject secrets into a template
// and write the result to a file.
func (sg *SecretGroup) PushToFile(secrets []*Secret) (bool, error) {
	return sg.pushToFileWithDeps(openFileAsWriteCloser, pushToWriter, secrets)
}

func (sg *SecretGroup) pushToFileWithDeps(
	depOpenWriteCloser openWriteCloserFunc,
	depPushToWriter pushToWriterFunc,
	secrets []*Secret,
) (updated bool, err error) {
	// Make sure all the secret specs are accounted for
	if err = validateSecretsAgainstSpecs(secrets, sg.SecretSpecs); err != nil {
		return false, err
	}

	// Determine file template from
	// 1. File template
	// 2. File format
	// 3. Secret specs (user to validate file template)
	fileTemplate, err := maybeFileTemplateFromFormat(
		sg.FileTemplate,
		sg.FileFormat,
		sg.SecretSpecs,
	)
	if err != nil {
		return false, err
	}

	//// Open and push to file
	wc, err := depOpenWriteCloser(sg.FilePath, sg.FilePermissions)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = wc.Close()
	}()

	maskError := fmt.Errorf("failed to execute template, with secret values, on push to file for secret group %q", sg.Name)
	defer func() {
		if r := recover(); r != nil {
			err = maskError
		}
	}()
	updated, err = depPushToWriter(
		wc,
		sg.Name,
		fileTemplate,
		secrets,
	)
	if err != nil {
		err = maskError
	}
	return updated, err
}

func (sg *SecretGroup) absoluteFilePath(secretsBasePath string) (string, error) {
	groupName := sg.Name
	filePath := sg.FilePath
	fileTemplate := sg.FileTemplate
	fileExt := sg.FileFormat

	// filePath must be relative
	if path.IsAbs(filePath) {
		return "", fmt.Errorf(
			"provided filepath %q for secret group %q is absolute, requires relative path",
			filePath, groupName,
		)
	}

	pathContainsFilename := !strings.HasSuffix(filePath, "/") && len(filePath) > 0

	if !pathContainsFilename {
		if len(fileTemplate) > 0 {
			// Template filename defaults to "{groupName}.out"
			fileExt = "out"
		}

		// For all other formats, the filename defaults to "{groupName}.{fileFormat}"
		filePath = path.Join(
			filePath,
			fmt.Sprintf("%s.%s", groupName, fileExt),
		)
		log.Info(messages.CSPFK017I, groupName)
	}

	absoluteFilePath := path.Join(secretsBasePath, filePath)

	// filePath must be relative to secrets base path. This protects against relative paths
	// that, by using the double-dot path segment, resolve to a path that is not relative
	// to the base path.
	if !strings.HasPrefix(absoluteFilePath, secretsBasePath) {
		return "", fmt.Errorf(
			"provided filepath %q for secret group %q must be relative to secrets base path",
			filePath, groupName,
		)
	}

	// Filename cannot be longer than allowed by the filesystem
	_, filename := path.Split(absoluteFilePath)
	if len(filename) > maxFilenameLen {
		return "", fmt.Errorf(
			"filename %q for provided filepath for secret group %q must not be longer than %d characters",
			filename,
			groupName,
			maxFilenameLen,
		)
	}

	return absoluteFilePath, nil
}

func (sg *SecretGroup) validate() []error {
	groupName := sg.Name
	fileFormat := sg.FileFormat
	fileTemplate := sg.FileTemplate
	secretSpecs := sg.SecretSpecs

	if errors := validateSecretPaths(secretSpecs, groupName); len(errors) > 0 {
		return errors
	}

	if len(fileFormat) > 0 && fileFormat != "template" {
		_, err := FileTemplateForFormat(fileFormat, secretSpecs)
		if err != nil {
			return []error{
				fmt.Errorf(
					"unable to process group %q into file format %q: %s",
					groupName,
					fileFormat,
					err,
				),
			}
		}
	}

	// First-pass at provided template rendering with dummy secret values
	// This first-pass is limited for templates that branch conditionally on secret values
	// Relying logically on specific secret values should be avoided
	if len(fileTemplate) > 0 {
		dummySecrets := []*Secret{}
		for _, secretSpec := range secretSpecs {
			dummySecrets = append(dummySecrets, &Secret{Alias: secretSpec.Alias, Value: "REDACTED"})
		}

		_, err := pushToWriter(ioutil.Discard, groupName, fileTemplate, dummySecrets)
		if err != nil {
			return []error{fmt.Errorf(
				`unable to use file template for secret group %q: %s`,
				groupName,
				err,
			)}
		}
	}

	return nil
}

func validateSecretsAgainstSpecs(
	secrets []*Secret,
	specs []SecretSpec,
) error {
	if len(secrets) != len(specs) {
		return fmt.Errorf(
			"number of secrets (%d) does not match number of secret specs (%d)",
			len(secrets),
			len(specs),
		)
	}

	// Secrets should match SecretSpecs
	var aliasInSecrets = map[string]struct{}{}
	for _, secret := range secrets {
		aliasInSecrets[secret.Alias] = struct{}{}
	}

	var missingAliases []string
	for _, spec := range specs {
		if _, ok := aliasInSecrets[spec.Alias]; !ok {
			missingAliases = append(missingAliases, spec.Alias)
		}
	}

	// Sort strings to ensure deterministic behavior of the method
	sort.Strings(missingAliases)

	if len(missingAliases) > 0 {
		return fmt.Errorf("some secret specs are not present in secrets %q", strings.Join(missingAliases, ""))
	}

	return nil
}

func maybeFileTemplateFromFormat(
	fileTemplate string,
	fileFormat string,
	secretSpecs []SecretSpec,
) (string, error) {
	// Default to "yaml" file format
	if len(fileTemplate)+len(fileFormat) == 0 {
		fileFormat = "yaml"
	}

	// fileFormat is used to set fileTemplate when fileTemplate is not
	// already set
	if len(fileTemplate) == 0 {
		var err error

		fileTemplate, err = FileTemplateForFormat(
			fileFormat,
			secretSpecs,
		)
		if err != nil {
			return "", err
		}
	}

	return fileTemplate, nil
}

// NewSecretGroups creates a collection of secret groups from a map of annotations
func NewSecretGroups(
	secretsBasePath string,
	templatesBasePath string,
	annotations map[string]string,
) ([]*SecretGroup, []error) {
	c := Config{
		secretsBasePath:   secretsBasePath,
		templatesBasePath: templatesBasePath,
		openReadCloser:    openFileAsReadCloser,
		pullFromReader:    pullFromReader,
	}

	return newSecretGroupsWithDeps(annotations, c)
}

func newSecretGroupsWithDeps(annotations map[string]string, c Config) ([]*SecretGroup, []error) {
	var sgs []*SecretGroup

	var errors []error
	for key := range annotations {
		if strings.HasPrefix(key, secretGroupPrefix) {
			groupName := strings.TrimPrefix(key, secretGroupPrefix)
			sg, errs := newSecretGroup(groupName, annotations, c)
			if errs != nil {
				errors = append(errors, errs...)
				continue
			}
			sgs = append(sgs, sg)
		}
	}

	errors = append(errors, validateGroupFilePaths(sgs)...)

	if len(errors) > 0 {
		return nil, errors
	}

	// Sort secret groups for deterministic order based on group path
	sort.SliceStable(sgs, func(i, j int) bool {
		return sgs[i].Name < sgs[j].Name
	})

	return sgs, nil
}

func newSecretGroup(groupName string, annotations map[string]string, c Config) (*SecretGroup, []error) {
	groupSecrets := annotations[secretGroupPrefix+groupName]
	filePath := annotations[secretGroupFilePathPrefix+groupName]
	fileFormat := annotations[secretGroupFileFormatPrefix+groupName]

	policyPathPrefix := annotations[secretGroupPolicyPathPrefix+groupName]
	policyPathPrefix = strings.TrimPrefix(policyPathPrefix, "/")
	filePermissions := annotations[secretGroupFilePermissionsPrefix+groupName]

	var err error
	var fileTemplate string
	if fileFormat == "template" {
		fileTemplate, err = collectTemplate(groupName, annotations, c)
		if err != nil {
			return nil, []error{err}
		}
	}

	// Default to "yaml" file format
	if len(fileFormat) == 0 {
		fileFormat = "yaml"
	}

	fileMode, err := permStrToFileMode(filePermissions)
	if err != nil {
		err = fmt.Errorf(`unable to create fileMode from annotation "%s": %s`, secretGroupFilePermissionsPrefix, err)
		return nil, []error{err}
	}

	secretSpecs, err := NewSecretSpecs([]byte(groupSecrets))
	if err != nil {
		err = fmt.Errorf(`unable to create secret specs from annotation "%s": %s`, secretGroupPrefix+groupName, err)
		return nil, []error{err}
	}
	secretSpecs = resolvedSecretSpecs(policyPathPrefix, secretSpecs)

	sg := &SecretGroup{
		Name:             groupName,
		FilePath:         filePath,
		FileTemplate:     fileTemplate,
		FileFormat:       fileFormat,
		FilePermissions:  *fileMode,
		PolicyPathPrefix: policyPathPrefix,
		SecretSpecs:      secretSpecs,
	}

	errors := sg.validate()
	if len(errors) > 0 {
		return nil, errors
	}

	// Generate absolute file path
	sg.FilePath, err = sg.absoluteFilePath(c.secretsBasePath)
	if err != nil {
		return nil, []error{err}
	}

	return sg, nil
}

func collectTemplate(groupName string, annotations map[string]string, c Config) (string, error) {
	annotationTemplate := annotations[secretGroupFileTemplatePrefix+groupName]

	configmapTemplate, err := readTemplateFromFile(groupName, annotations, c)
	if os.IsNotExist(err) {
		return annotationTemplate, nil
	} else if err != nil {
		return "", fmt.Errorf("unable to read template file for secret group %q: %s", groupName, err)
	}

	if len(annotationTemplate) > 0 && len(configmapTemplate) > 0 {
		return "", fmt.Errorf("secret file template for group %q cannot be provided both by annotation and by ConfigMap", groupName)
	}

	if len(annotationTemplate)+len(configmapTemplate) == 0 {
		return "", fmt.Errorf("template required for secret group %q", groupName)
	}

	return configmapTemplate, nil
}

func readTemplateFromFile(
	groupName string,
	annotations map[string]string,
	c Config,

) (string, error) {
	templateFilepath := filepath.Join(c.templatesBasePath, groupName+".tpl")
	rc, err := c.openReadCloser(templateFilepath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = rc.Close()
	}()

	return c.pullFromReader(rc)
}

func permStrToFileMode(perms string) (*os.FileMode, error) {
	if perms == "" {
		fileMode := defaultFilePermissions
		return &fileMode, nil
	}

	invalidFormatErr := fmt.Errorf("Invalid permissions format: '%s'", perms)
	// Permissions string should be 9 digits or 10 digits with leading '-'
	switch len(perms) {
	case 9:
		break
	case 10:
		if perms[0] != '-' {
			return nil, invalidFormatErr
		}
		perms = perms[1:]
	default:
		return nil, invalidFormatErr
	}

	invalidPermissionsErr := fmt.Errorf("Invalid permissions: '%s', owner permissions must atleast have read and write (-rw-------)", perms)
	//User Group should atleast have read/write permissions
	if perms[0] != 'r' || perms[1] != 'w' {
		return nil, invalidPermissionsErr
	}

	validChars := "rwx"
	multipliers := []int{64, 8, 1}
	bitValues := []int{4, 2, 1}

	result := 0
	index := 0
	for group := 0; group < 3; group++ {
		for bit := 0; bit < 3; bit++ {
			switch perms[index] {
			case validChars[bit]:
				result += bitValues[bit] * multipliers[group]
			case '-':
				break
			default:
				return nil, invalidFormatErr
			}
			index++
		}
	}

	fileMode := os.FileMode(result)
	return &fileMode, nil
}

func validateGroupFilePaths(secretGroups []*SecretGroup) []error {
	// Iterate over the secret groups and group any that have the same file path
	groupFilePaths := make(map[string][]string)
	for _, sg := range secretGroups {
		if len(groupFilePaths[sg.FilePath]) == 0 {
			groupFilePaths[sg.FilePath] = []string{sg.Name}
			continue
		}

		groupFilePaths[sg.FilePath] = append(groupFilePaths[sg.FilePath], sg.Name)
	}

	// If any file paths are used in more than one group, log all the groups that share the path
	var errors []error
	for path, groupNames := range groupFilePaths {
		if len(groupNames) > 1 {
			errors = append(errors, fmt.Errorf(
				"duplicate filepath %q for groups: %q", path, strings.Join(groupNames, `, `),
			))
		}
	}
	return errors
}
