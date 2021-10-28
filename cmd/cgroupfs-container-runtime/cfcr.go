package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/openkhal/cgroupfs/pkg/oci"
	log "github.com/sirupsen/logrus"
)

type cgroupfsContainerRuntime struct {
	logger  *log.Logger
	runtime oci.Runtime
	ociSpec oci.Spec
}

var _ oci.Runtime = (*cgroupfsContainerRuntime)(nil)

// newNvidiaContainerRuntime is a constructor for a standard runtime shim.
func newCgroupfsContainerRuntimeWithLogger(logger *log.Logger, runtime oci.Runtime, ociSpec oci.Spec) (oci.Runtime, error) {
	r := cgroupfsContainerRuntime{
		logger:  logger,
		runtime: runtime,
		ociSpec: ociSpec,
	}

	return &r, nil
}

func (r cgroupfsContainerRuntime) Exec(args []string) error {
	if r.modificationRequired(args) {
		err := r.modifyOCISpec()
		if err != nil {
			return fmt.Errorf("error modifying OCI spec: %v", err)
		}
	}

	r.logger.Println("Forwarding command to runtime")
	return r.runtime.Exec(args)
}

// modificationRequired checks the intput arguments to determine whether a modification
// to the OCI spec is required.
func (r cgroupfsContainerRuntime) modificationRequired(args []string) bool {
	var previousWasBundle bool
	for _, a := range args {
		// We check for '--bundle create' explicitly to ensure that we
		// don't inadvertently trigger a modification if the bundle directory
		// is specified as `create`
		if !previousWasBundle && isBundleFlag(a) {
			previousWasBundle = true
			continue
		}

		if !previousWasBundle && a == "create" {
			r.logger.Infof("'create' command detected; modification required")
			return true
		}

		previousWasBundle = false
	}

	r.logger.Infof("No modification required")
	return false
}

func (r cgroupfsContainerRuntime) modifyOCISpec() error {
	err := r.ociSpec.Load()
	if err != nil {
		return fmt.Errorf("error loading OCI specification for modification: %v", err)
	}

	err = r.ociSpec.Modify(r.addCgroupfsHook)
	if err != nil {
		return fmt.Errorf("error injecting Cgroufs Container Runtime hook: %v", err)
	}

	err = r.ociSpec.Flush()
	if err != nil {
		return fmt.Errorf("error writing modified OCI specification: %v", err)
	}
	return nil
}

func (r cgroupfsContainerRuntime) addCgroupfsHook(spec *specs.Spec) error {
	path, err := exec.LookPath("cgroupfs-container-runtime-hook")
	if err != nil {
		path = hookDefaultFilePath
		_, err = os.Stat(path)
		if err != nil {
			return err
		}
	}

	r.logger.Printf("prestart hook path: %s\n", path)

	args := []string{path}
	if spec.Hooks == nil {
		spec.Hooks = &specs.Hooks{}
	} else if len(spec.Hooks.Prestart) != 0 {
		for _, hook := range spec.Hooks.Prestart {
			if !strings.Contains(hook.Path, "cgroupfs-container-runtime-hook") {
				continue
			}
			r.logger.Println("existing cgroupfs prestart hook in OCI spec file")
			return nil
		}
	}

	spec.Hooks.Prestart = append(spec.Hooks.Prestart, specs.Hook{
		Path: path,
		Args: append(args, "prestart"),
	})

	return nil
}
