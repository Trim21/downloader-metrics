package reporter

import (
	"context"
	"time"

	"github.com/hekmon/transmissionrpc/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	"app/pkg/transmission"
	"app/pkg/utils"
)

var torrentFields = []string{"hashString", "status", "name", "labels", "uploadedEver", "downloadedEver"}

func setupTransmissionMetrics() (prometheus.Collector, error) {
	client, err := transmission.New()
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, nil
	}

	log.Info().Msg("enable transmission reporter")

	return transmissionExporter{client: client}, nil
}

type transmissionExporter struct {
	client *transmissionrpc.Client
}

func (r transmissionExporter) Describe(c chan<- *prometheus.Desc) {
}

func (r transmissionExporter) Collect(m chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	h, err := r.client.SessionStats(ctx)
	if err != nil {
		log.Err(err).Msg("failed to fetch transmission server stats")
		return
	}

	m <- utils.Gauge("transmission_download_session_bytes", nil, float64(h.CurrentStats.DownloadedBytes))
	m <- utils.Gauge("transmission_upload_session_bytes", nil, float64(h.CurrentStats.UploadedBytes))
	m <- utils.Gauge("transmission_download_total_bytes", nil, float64(h.CumulativeStats.DownloadedBytes))
	m <- utils.Gauge("transmission_upload_total_bytes", nil, float64(h.CumulativeStats.UploadedBytes))

	torrents, err := r.client.TorrentGet(ctx, torrentFields, nil)
	if err != nil {
		log.Err(err).Msg("failed to fetch transmission torrents")
		return
	}

	for _, torrent := range torrents {
		label := prometheus.Labels{"hash": *torrent.HashString}
		m <- utils.Gauge("transmission_torrent_upload_bytes", label, float64(*torrent.UploadedEver))
		m <- utils.Gauge("transmission_torrent_download_bytes", label, float64(*torrent.DownloadedEver))
	}
}
