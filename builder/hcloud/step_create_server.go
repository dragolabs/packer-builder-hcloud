package hcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer/packer"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/mitchellh/multistep"
)

type stepCreateServer struct {
	serverID int
}

func (s *stepCreateServer) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*hcloud.Client)
	config := state.Get("config").(Config)
	ui := state.Get("ui").(packer.Ui)

	ctx := context.Background()

	serverType, _, err := client.ServerType.Get(ctx, config.ServerType)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	sourceImage, _, err := client.Image.Get(ctx, config.SourceImage)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	sshKeyID := state.Get("ssh_key_id").(int)

	sshKey, _, err := client.SSHKey.GetByID(ctx, sshKeyID)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	name := fmt.Sprintf("packer-hcloud-%d", time.Now().UnixNano())
	ui.Say(fmt.Sprintf("Creating new server: %s", name))

	serverCreateOpts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: serverType,
		Image:      sourceImage,
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		UserData:   config.UserData,
	}

	if config.Location != "" {
		location, _, err := client.Location.GetByName(ctx, config.Location)
		if err != nil {
			ui.Error(err.Error())
			state.Put("error", err)
			return multistep.ActionHalt
		}

		serverCreateOpts.Location = location
	}

	if config.Datacenter != "" {
		datacenter, _, err := client.Datacenter.GetByName(ctx, config.Datacenter)
		if err != nil {
			ui.Error(err.Error())
			state.Put("error", err)
			return multistep.ActionHalt
		}

		serverCreateOpts.Datacenter = datacenter
	}

	serverData, _, err := client.Server.Create(ctx, serverCreateOpts)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	state.Put("server_data", serverData)
	s.serverID = serverData.Server.ID

	ui.Say(fmt.Sprintf("Created server %d", s.serverID))

	return multistep.ActionContinue
}

func (s *stepCreateServer) Cleanup(state multistep.StateBag) {
	client := state.Get("client").(*hcloud.Client)
	ui := state.Get("ui").(packer.Ui)

	if s.serverID <= 0 {
		return
	}

	ui.Say(fmt.Sprintf("Waiting for server %d to be destroyed...", s.serverID))

	ctx := context.Background()

	server, _, err := client.Server.GetByID(ctx, s.serverID)

	_, err = client.Server.Delete(ctx, server)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return
	}
}
