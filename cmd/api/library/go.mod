module github.com/ellypaws/inkbunny-app/api/library

replace github.com/ellypaws/inkbunny-app/api => ../

go 1.22.1

require (
	github.com/disintegration/imaging v1.6.2
	github.com/ellypaws/inkbunny-app/api v0.0.0
	github.com/ellypaws/inkbunny-sd v0.0.0-20240413140044-10290d136096
	github.com/ellypaws/inkbunny/api v0.0.0-20240411110242-d491ced97f23
	github.com/stretchr/testify v1.9.0
	golang.org/x/net v0.24.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/image v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
