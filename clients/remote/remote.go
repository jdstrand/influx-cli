package remote

import (
	"context"
	"errors"
	"fmt"

	"github.com/influxdata/influx-cli/v2/api"
	"github.com/influxdata/influx-cli/v2/clients"
)

type Client struct {
	clients.CLI
	api.RemoteConnectionsApi
	api.OrganizationsApi
}

type CreateParams struct {
	clients.OrgParams
	Name             string
	Description      string
	RemoteURL        string
	RemoteAPIToken   string
	RemoteOrgID      string
	AllowInsecureTLS bool
}

func (c Client) Create(ctx context.Context, params *CreateParams) error {

	orgID, err := params.GetOrgID(ctx, c.ActiveConfig, c.OrganizationsApi)
	if err != nil {
		return err
	}

	// set up a struct with required params
	body := api.RemoteConnectionCreationRequest{
		Name:             params.Name,
		OrgID:            orgID,
		RemoteURL:        params.RemoteURL,
		RemoteAPIToken:   params.RemoteAPIToken,
		RemoteOrgID:      params.RemoteOrgID,
		AllowInsecureTLS: params.AllowInsecureTLS,
	}

	if params.Description != "" {
		body.Description = &params.Description
	}

	// send post request
	res, err := c.PostRemoteConnection(ctx).RemoteConnectionCreationRequest(body).Execute()
	if err != nil {
		return fmt.Errorf("failed to create remote connection %q: %w", params.Name, err)
	}

	// print confirmation of new connection
	return c.printRemote(printRemoteOpts{remote: &res})
}

type UpdateParams struct {
	RemoteID         string
	Name             string
	Description      string
	RemoteURL        string
	RemoteAPIToken   string
	RemoteOrgID      string
	AllowInsecureTLS bool
	TLSFlagIsSet     bool
}

func (c Client) Update(ctx context.Context, params *UpdateParams) error {
	// build request
	body := api.RemoteConnenctionUpdateRequest{}

	if params.Name != "" {
		body.SetName(params.Name)
	}

	if params.Description != "" {
		body.SetDescription(params.Description)
	}

	if params.RemoteURL != "" {
		body.SetRemoteURL(params.RemoteURL)
	}

	if params.RemoteAPIToken != "" {
		body.SetRemoteAPIToken(params.RemoteAPIToken)
	}

	if params.RemoteOrgID != "" {
		body.SetRemoteOrgID(params.RemoteOrgID)
	}

	if params.TLSFlagIsSet {
		body.SetAllowInsecureTLS(params.AllowInsecureTLS)
	}

	// send patch request
	res, err := c.PatchRemoteConnectionByID(ctx, params.RemoteID).RemoteConnenctionUpdateRequest(body).Execute()
	if err != nil {
		return fmt.Errorf("failed to update remote connection %q: %w", params.RemoteID, err)
	}
	// print updated remote connection info
	return c.printRemote(printRemoteOpts{remote: &res})
}

type ListParams struct {
	clients.OrgParams
	Name      string
	RemoteURL string
}

func (c Client) List(ctx context.Context, params *ListParams) error {

	orgID, err := params.GetOrgID(ctx, c.ActiveConfig, c.OrganizationsApi)
	if err != nil {
		return err
	}

	// set up query params
	req := c.GetRemoteConnections(ctx).OrgID(orgID)

	if params.Name != "" {
		req = req.Name(params.Name)
	}

	if params.RemoteURL != "" {
		req = req.RemoteURL(params.RemoteURL)
	}

	// send get request
	res, err := req.Execute()
	if err != nil {
		return fmt.Errorf("failed to get remote connections: %w", err)
	}

	// print connections
	printOpts := printRemoteOpts{}
	if res.Remotes != nil {
		printOpts.remotes = *res.Remotes
	} else {
		return errors.New("no remote connections found for specified parameters")
	}

	return c.printRemote(printOpts)
}

func (c Client) Delete(ctx context.Context, remoteID string) error {
	connection, err := c.GetRemoteConnectionByID(ctx, remoteID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete remote connection %q: %w", remoteID, err)
	}

	req := c.DeleteRemoteConnectionByID(ctx, remoteID)

	// send delete request
	err = req.Execute()
	if err != nil {
		return fmt.Errorf("failed to delete remote connection %q: %w", remoteID, err)
	}

	// print deleted connection info
	printOpts := printRemoteOpts{
		remote:  &connection,
		deleted: true,
	}

	return c.printRemote(printOpts)
}

type printRemoteOpts struct {
	remote  *api.RemoteConnection
	remotes []api.RemoteConnection
	deleted bool
}

func (c Client) printRemote(opts printRemoteOpts) error {
	if c.PrintAsJSON {
		var v interface{}
		if opts.remote != nil {
			v = opts.remote
		} else {
			v = opts.remotes
		}
		return c.PrintJSON(v)
	}

	headers := []string{"ID", "Name", "Org ID", "Remote URL", "Remote Org ID", "Allow Insecure TLS"}
	if opts.deleted {
		headers = append(headers, "Deleted")
	}

	if opts.remote != nil {
		opts.remotes = append(opts.remotes, *opts.remote)
	}

	var rows []map[string]interface{}
	for _, r := range opts.remotes {
		row := map[string]interface{}{
			"ID":                 r.GetId(),
			"Name":               r.GetName(),
			"Org ID":             r.OrgID,
			"Remote URL":         r.RemoteURL,
			"Remote Org ID":      r.RemoteOrgID,
			"Allow Insecure TLS": r.AllowInsecureTLS,
		}
		if opts.deleted {
			row["Deleted"] = true
		}
		rows = append(rows, row)
	}

	return c.PrintTable(headers, rows...)
}
