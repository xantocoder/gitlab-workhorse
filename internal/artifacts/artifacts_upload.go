package artifacts

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	//"github.com/gabriel-vasile/mimetype"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/api"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/filestore"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/helper"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/upload"
	"gitlab.com/gitlab-org/gitlab-workhorse/internal/zipartifacts"
)

var zipSubcommandsErrorsCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gitlab_workhorse_zip_subcommand_errors_total",
		Help: "Errors comming from subcommands used for processing ZIP archives",
	}, []string{"error"})

func init() {
	prometheus.MustRegister(zipSubcommandsErrorsCounter)
}

type artifactsUploadProcessor struct {
	opts *filestore.SaveFileOpts

	upload.SavedFileTracker
}

type storeFileResult struct {
	handler *filestore.FileHandler
	err     error
}

func (a *artifactsUploadProcessor) generateMetadataFromZip(ctx context.Context, file *filestore.FileHandler) (*filestore.FileHandler, error) {
	metaReader, metaWriter := io.Pipe()
	defer metaWriter.Close()

	metaOpts := &filestore.SaveFileOpts{
		LocalTempPath:  a.opts.LocalTempPath,
		TempFilePrefix: "metadata.gz",
	}
	if metaOpts.LocalTempPath == "" {
		metaOpts.LocalTempPath = os.TempDir()
	}

	fileName := file.LocalPath
	if fileName == "" {
		fileName = file.RemoteURL
	}

	// mime, err := mimetype.DetectFile(fileName)
	// if err != nil {
	// }

	done := make(chan storeFileResult)
	go func() {
		handler, err := filestore.SaveFileFromReader(ctx, metaReader, -1, metaOpts)

		done <- storeFileResult{handler: handler, err: err}
	}()

	err := a.processZipMetadata(ctx, fileName, metaWriter)
	if err != nil {
		return nil, err
	}

	metaWriter.Close()
	result := <-done
	return result.handler, result.err
}

func (a *artifactsUploadProcessor) processZipMetadata(ctx context.Context, file string, writer io.Writer) error {
	zipMd := exec.CommandContext(ctx, "gitlab-zip-metadata", file)
	zipMd.Stderr = log.ContextLogger(ctx).Writer()
	zipMd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	zipMd.Stdout = writer

	if err := zipMd.Start(); err != nil {
		return err
	}
	defer helper.CleanUpProcessGroup(zipMd)

	err := zipMd.Wait()
	if err != nil {
		st, ok := helper.ExitStatus(err)

		if !ok {
			return err
		}

		zipSubcommandsErrorsCounter.WithLabelValues(zipartifacts.ErrorLabelByCode(st)).Inc()

		if st == zipartifacts.CodeNotZip {
			return nil
		}

		if st == zipartifacts.CodeLimitsReached {
			return zipartifacts.ErrBadMetadata
		}
	}

	return err
}

func (a *artifactsUploadProcessor) ProcessFile(ctx context.Context, formName string, file *filestore.FileHandler, writer *multipart.Writer) error {
	//  ProcessFile for artifacts requires file form-data field name to eq `file`

	if formName != "file" {
		return fmt.Errorf("invalid form field: %q", formName)
	}
	if a.Count() > 0 {
		return fmt.Errorf("artifacts request contains more than one file")
	}
	a.Track(formName, file.LocalPath)

	select {
	case <-ctx.Done():
		return fmt.Errorf("ProcessFile: context done")

	default:
		// TODO: can we rely on disk for shipping metadata? Not if we split workhorse and rails in 2 different PODs
		metadata, err := a.generateMetadataFromZip(ctx, file)
		if err != nil {
			return err
		}

		if metadata != nil {
			fields, err := metadata.GitLabFinalizeFields("metadata")
			if err != nil {
				return fmt.Errorf("finalize metadata field error: %v", err)
			}

			for k, v := range fields {
				writer.WriteField(k, v)
			}

			a.Track("metadata", metadata.LocalPath)
		}
	}

	return nil
}

func (a *artifactsUploadProcessor) Name() string {
	return "artifacts"
}

func UploadArtifacts(myAPI *api.API, h http.Handler, p upload.Preparer) http.Handler {
	return myAPI.PreAuthorizeHandler(func(w http.ResponseWriter, r *http.Request, a *api.Response) {
		opts, _, err := p.Prepare(a)
		if err != nil {
			helper.Fail500(w, r, fmt.Errorf("UploadArtifacts: error preparing file storage options"))
			return
		}

		mg := &artifactsUploadProcessor{opts: opts, SavedFileTracker: upload.SavedFileTracker{Request: r}}
		upload.HandleFileUploads(w, r, h, a, mg, opts)
	}, "/authorize")
}
