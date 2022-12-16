This directory will be used to define a package that provides generic parsing
of Annotations files that have been created by Kubernetes via the Kubernetes
Downward API. This package will read in files that contain entries of the form
`<key>`=`<value>`, and will create a Golang map of the form map[string]string.
