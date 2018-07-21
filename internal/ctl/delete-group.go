package ctl

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

// DeleteGroupCmd deletes a group
type DeleteGroupCmd struct {
	name        string
	displayName string
	gid         int
}

// Name returns the name of this cmdlet.
func (*DeleteGroupCmd) Name() string { return "delete-group" }

// Synopsis returns the short-form info for this cmdlet.
func (*DeleteGroupCmd) Synopsis() string { return "Delete a group existing on the server." }

// Usage returns the long-form info form this cmdlet.
func (*DeleteGroupCmd) Usage() string {
	return `new-group --name <name>
Delete the named group.
`
}

// SetFlags is the interface function which sets flags specific to this cmdlet.
func (p *DeleteGroupCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.name, "name", "", "Name for the new group.")
}

// Execute is the interface function which runs this cmdlet.
func (p *DeleteGroupCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Grab a client
	c, err := getClient()
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	// Get the authorization token
	t, err := getToken(c, getEntity())
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	result, err := c.DeleteGroup(p.name, t)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	fmt.Println(result.GetMsg())
	return subcommands.ExitSuccess
}
