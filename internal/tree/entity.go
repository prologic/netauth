package tree

import (
	"fmt"
	"log"
	"strings"

	"github.com/NetAuth/NetAuth/internal/tree/errors"
	"github.com/golang/protobuf/proto"

	pb "github.com/NetAuth/Protocol"
)

// NewEntity creates a new entity given an ID, number, and secret.
// Its not necessary to set the secret upon creation and it can be set
// later.  If not set on creation then the entity will not be usable.
// number must be a unique positive integer.  Because these are
// generally allocated in sequence the special value '-1' may be
// specified which will select the next available number.
func (m *Manager) NewEntity(ID string, number int32, secret string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID:     &ID,
			Number: &number,
			Secret: &secret,
		},
	}

	if err := ep.FetchHooks("CREATE", m.entityProcesses); err != nil {
		return err
	}
	_, err := ep.Run()
	return err
}

// MakeBootstrap is a function that can be called during the startup
// of the srever to create an entity that has the appropriate
// authority to create more entities and otherwise manage the server.
// This can only be called once during startup, attepts to call it
// again will result in no change.  The bootstrap user will always get
// the next available number which in most cases will be 1.
func (m *Manager) MakeBootstrap(ID string, secret string) {
	if m.bootstrapDone {
		return
	}

	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID:     &ID,
			Secret: &secret,
			Meta: &pb.EntityMeta{
				Capabilities: []pb.Capability{pb.Capability_GLOBAL_ROOT},
			},
		},
	}

	if err := ep.FetchHooks("BOOTSTRAP-SERVER", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	m.bootstrapDone = true

	if err != nil {
		log.Println("Bootstrap FAILED:")
		log.Fatal(err)
	}
}

// DisableBootstrap disables the ability to bootstrap after the
// opportunity to do so has passed.
func (m *Manager) DisableBootstrap() {
	log.Println("Disabling bootstrap")
	m.bootstrapDone = true
	log.Println("Bootstrap disabled")
}

// DeleteEntityByID deletes the named entity.  This function will
// delete the entity in a non-atomic way, but will ensure that the
// entity cannot be authenticated with before returning.  If the named
// ID does not exist the function will return tree.E_NO_ENTITY, in
// all other cases nil is returned.
func (m *Manager) DeleteEntityByID(ID string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID: &ID,
		},
	}

	if err := ep.FetchHooks("DESTROY", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// SetEntityCapabilityByID adds a capability to an entry directly.
func (m *Manager) SetEntityCapabilityByID(ID string, c string) error {
	capIndex, ok := pb.Capability_value[c]
	if !ok {
		return tree.ErrUnknownCapability
	}

	ep := EntityProcessor{
		Entity: &pb.Entity{
			ID: &ID,
		},
		RequestData: &pb.Entity{
			Meta: &pb.EntityMeta{
				Capabilities: []pb.Capability{pb.Capability(capIndex)},
			},
		},
	}

	if err := ep.FetchHooks("SET-CAPABILITY", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// RemoveEntityCapabilityByID is a convenience function to get the entity
// and hand it off to the actual removeEntityCapability function
func (m *Manager) RemoveEntityCapabilityByID(ID string, c string) error {
	capIndex, ok := pb.Capability_value[c]
	if !ok {
		return tree.ErrUnknownCapability
	}

	ep := EntityProcessor{
		Entity: &pb.Entity{
			ID: &ID,
		},
		RequestData: &pb.Entity{
			Meta: &pb.EntityMeta{
				Capabilities: []pb.Capability{pb.Capability(capIndex)},
			},
		},
	}

	if err := ep.FetchHooks("DROP-CAPABILITY", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// SetEntitySecretByID sets the secret on a given entity using the
// crypto interface.
func (m *Manager) SetEntitySecretByID(ID string, secret string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID:     &ID,
			Secret: &secret,
		},
	}

	if err := ep.FetchHooks("SET-SECRET", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// ValidateSecret validates the identity of an entity by
// validating the authenticating entity with the secret.
func (m *Manager) ValidateSecret(ID string, secret string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID:     &ID,
			Secret: &secret,
		},
	}

	if err := ep.FetchHooks("VALIDATE-IDENTITY", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// GetEntity returns an entity to the caller after first making a safe
// copy of it to remove secure fields.
func (m *Manager) GetEntity(ID string) (*pb.Entity, error) {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID: &ID,
		},
	}

	if err := ep.FetchHooks("FETCH", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	e, err := ep.Run()
	if err != nil {
		return nil, err
	}

	// The safeCopyEntity will return the entity without secrets
	// in it, as well as an error if there were problems
	// marshaling the proto back and forth.
	return safeCopyEntity(e), nil
}

// UpdateEntityMeta drives the internal version by obtaining the
// entity from the database based on the ID.
func (m *Manager) UpdateEntityMeta(ID string, newMeta *pb.EntityMeta) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID:   &ID,
			Meta: newMeta,
		},
	}

	if err := ep.FetchHooks("MERGE-METADATA", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// updateEntityKeys performs an update on keys to allow the client to
// be simpler, and to account for proto.Merge() merging list contents
// rather than overwriting.
func (m *Manager) updateEntityKeys(e *pb.Entity, mode, keyType, key string) ([]string, error) {
	// Normalize the type and the mode
	mode = strings.ToUpper(mode)
	keyType = strings.ToUpper(keyType)

	if e.Meta == nil {
		e.Meta = &pb.EntityMeta{}
	}

	switch mode {
	case "LIST":
		return e.GetMeta().GetKeys(), nil
	case "ADD":
		e.Meta.Keys = patchStringSlice(e.Meta.Keys, fmt.Sprintf("%s:%s", keyType, key), true, true)
	case "DEL":
		e.Meta.Keys = patchStringSlice(e.Meta.Keys, key, false, false)
	}

	// Save changes
	if err := m.db.SaveEntity(e); err != nil {
		return nil, err
	}
	return nil, nil
}

// UpdateEntityKeys is the exported version of updateEntityKeys
func (m *Manager) UpdateEntityKeys(entityID, mode, keytype, key string) ([]string, error) {
	e, err := m.db.LoadEntity(entityID)
	if err != nil {
		return nil, err
	}

	return m.updateEntityKeys(e, mode, keytype, key)
}

// ManageUntypedEntityMeta handles the things that may be annotated
// onto an entity.  These annotations should be used sparingly as they
// incur a non-trivial lookup cost on the server.
func (m *Manager) ManageUntypedEntityMeta(entityID, mode, key, value string) ([]string, error) {
	// Load Entity
	e, err := m.db.LoadEntity(entityID)
	if err != nil {
		return nil, err
	}

	if e.Meta == nil {
		e.Meta = &pb.EntityMeta{}
	}

	// Patch the KV slice
	tmp := patchKeyValueSlice(e.GetMeta().UntypedMeta, mode, key, value)

	// If this was a read, bail out now with whatever was read
	if strings.ToUpper(mode) == "READ" {
		return tmp, nil
	}

	// Save changes
	e.Meta.UntypedMeta = tmp
	if err := m.db.SaveEntity(e); err != nil {
		return nil, err
	}
	return nil, nil
}

// LockEntity allows external callers to lock entities directly.
// Internal users can just set the value directly.
func (m *Manager) LockEntity(ID string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID: &ID,
		},
	}

	if err := ep.FetchHooks("LOCK", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// UnlockEntity allows external callers to lock entities directly.
// Internal users can just set the value directly.
func (m *Manager) UnlockEntity(ID string) error {
	ep := EntityProcessor{
		Entity: &pb.Entity{},
		RequestData: &pb.Entity{
			ID: &ID,
		},
	}

	if err := ep.FetchHooks("UNLOCK", m.entityProcesses); err != nil {
		log.Fatal(err)
	}
	_, err := ep.Run()
	return err
}

// safeCopyEntity makes a copy of the entity provided but removes
// fields that are related to security.  This permits the entity that
// is returned to be handed off outside the server.
func safeCopyEntity(e *pb.Entity) *pb.Entity {
	dup := &pb.Entity{}
	proto.Merge(dup, e)

	// Fields for security are nulled out before returning.
	dup.Secret = proto.String("<REDACTED>")

	return dup
}
