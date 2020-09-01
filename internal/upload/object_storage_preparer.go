package upload

import (
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/config"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/filestore"
)

type ObjectStoragePreparer struct {
	credentials config.ObjectStorageCredentials
}

func NewObjectStoragePreparer(c config.Config) Preparer {
	creds := c.ObjectStorageCredentials

	if creds == nil {
		creds = &config.ObjectStorageCredentials{}
	}

	return &ObjectStoragePreparer{credentials: *creds}
}

func (p *ObjectStoragePreparer) Prepare(a *api.Response) (*filestore.SaveFileOpts, Verifier, error) {
	opts, err := filestore.GetOpts(a)
	if err != nil {
		return nil, nil, err
	}

	config := &opts.ObjectStorageConfig
	config.AzureCredentials = p.credentials.AzureCredentials
	config.S3Credentials = p.credentials.S3Credentials

	if err := config.RegisterGoCloudURLOpeners(); err != nil {
		log.WithError(err).Warn("unable to register GoCloud mux")
		return nil, nil, err
	}

	return opts, nil, nil
}
