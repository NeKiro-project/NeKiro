package file

import (
	"context"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/config_center/testkit"
)

func TestFileReadOnlyConformance(t *testing.T) {
	testkit.Run(t, newReadOnlyHarness(t))
}

func TestFileReaderCloseConformance(t *testing.T) {
	testkit.RunReaderClose(t, newReadOnlyHarness(t))
}

func TestFilePublisherConformance(t *testing.T) {
	testkit.Run(t, newPublisherHarness(t))
}

func newReadOnlyHarness(t *testing.T) testkit.Harness {
	t.Helper()
	reader, err := OpenReader(ReaderConfig{
		Root:               t.TempDir(),
		MaxPayloadBytes:    1024,
		SubscriptionBuffer: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, exported := any(reader).(interface {
		Interrupt(context.Context) error
	}); exported {
		t.Fatal("Reader exposes a production interruption control")
	}
	return testkit.Harness{
		Reader: reader,
		Interrupt: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			// Package-local test fixture control only: production callers cannot
			// name or invoke Reader.interruptAll.
			reader.interruptAll("")
			return nil
		},
		Cleanup: func() error {
			return reader.Close()
		},
	}
}

func newPublisherHarness(t *testing.T) testkit.Harness {
	t.Helper()
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{
		Root:               root,
		MaxPayloadBytes:    1024,
		SubscriptionBuffer: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := OpenPublisher(PublisherConfig{
		Root:            root,
		MaxPayloadBytes: 1024,
		FileMode:        testPublisherMode(),
	})
	if err != nil {
		_ = reader.Close()
		t.Fatal(err)
	}
	return testkit.Harness{
		Reader:    reader,
		Publisher: publisher,
		Publish:   publisher.Publish,
		Delete:    publisher.Delete,
		Interrupt: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			reader.interruptAll("")
			return nil
		},
		Cleanup: func() error {
			readerErr := reader.Close()
			publisherErr := publisher.Close()
			if readerErr != nil {
				return readerErr
			}
			return publisherErr
		},
	}
}

var _ configcenter.DynamicConfiguration = (*Reader)(nil)
