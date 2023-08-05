package controllers

import (
	"io/ember-csi-manager/controllers/emberstoragebackend"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, emberstoragebackend.Add)
}
