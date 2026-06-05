package cli

import (
	"context"
	"errors"
	"io"

	"github.com/jass/mojify/packages/core/internal/doctor"
)

func RunDoctor(ctx context.Context, stdout io.Writer) error {
	return runDoctorWithOptions(ctx, stdout, doctor.Options{})
}

func runDoctorWithOptions(ctx context.Context, stdout io.Writer, options doctor.Options) error {
	report := doctor.Run(ctx, options)
	doctor.Write(stdout, report)
	if report.Interrupted {
		return context.Canceled
	}
	if !report.OK() {
		return errors.New("required runtime tools are missing or unhealthy")
	}
	return nil
}
