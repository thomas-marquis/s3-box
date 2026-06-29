package directory_test

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func ExamplePath_RelativeTo() {
	path := directory.NewPath("/home/user/projects/src/controller")
	base := directory.NewPath("/home/user/projects/")

	res, err := path.RelativeTo(base)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.String())
	// Output:
	// src/controller/
}
