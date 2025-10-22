module github.com/tendant/simple-content/examples/photo-gallery

go 1.25.1

replace github.com/tendant/simple-content => ../..

require (
	github.com/google/uuid v1.6.0
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/tendant/simple-content v0.0.0-00010101000000-000000000000
)

require github.com/go-chi/chi/v5 v5.2.1 // indirect
