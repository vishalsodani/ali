package gui

import (
	"context"
	"fmt"
	"time"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/text"

	"github.com/nakabonne/ali/attacker"
)

type drawer struct {
	widgets   *widgets
	chartCh   chan *attacker.Result
	gaugeCh   chan bool
	metricsCh chan *attacker.Metrics

	// aims to avoid to perform multiple `redrawChart`.
	chartDrawing bool
}

// redrawChart appends entities as soon as a result arrives.
// Given maxSize, then it can be pre-allocated.
// TODO: In the future, multiple charts including bytes-in/out etc will be re-drawn.
func (d *drawer) redrawChart(ctx context.Context, maxSize int) {
	values := make([]float64, 0, maxSize)
	d.chartDrawing = true
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case res := <-d.chartCh:
			if res == nil {
				continue
			}
			if res.End {
				d.gaugeCh <- true
				break L
			}
			d.gaugeCh <- false
			values = append(values, float64(res.Latency/time.Millisecond))
			d.widgets.latencyChart.Series("latency", values,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(87))),
				linechart.SeriesXLabels(map[int]string{
					0: "req",
				}),
			)
		}
	}
	d.chartDrawing = false
}

func (d *drawer) redrawGauge(ctx context.Context, maxSize int) {
	var count float64
	size := float64(maxSize)
	d.widgets.progressGauge.Percent(0)
	for {
		select {
		case <-ctx.Done():
			return
		case end := <-d.gaugeCh:
			if end {
				return
			}
			count++
			d.widgets.progressGauge.Percent(int(count / size * 100))
		}
	}
}

const (
	latenciesTextFormat = `Total: %v
Mean: %v
P50: %v
P90: %v
P95: %v
P99: %v
Max: %v
Min: %v`

	bytesTextFormat = `In:
  Total: %v
  Mean: %v
Out:
  Total: %v
  Mean: %v`

	othersTextFormat = `Duration: %v
Wait: %v
Requests: %d
Rate: %f
Throughput: %f
Success: %f
Earliest: %v
Latest: %v
End: %v`
)

func (d *drawer) redrawMetrics(ctx context.Context) {
	defer close(d.metricsCh)
	for {
		select {
		case <-ctx.Done():
			return
		case metrics := <-d.metricsCh:
			if metrics == nil {
				continue
			}
			d.widgets.latenciesText.Write(
				fmt.Sprintf(latenciesTextFormat,
					metrics.Latencies.Total,
					metrics.Latencies.Mean,
					metrics.Latencies.P50,
					metrics.Latencies.P90,
					metrics.Latencies.P95,
					metrics.Latencies.P99,
					metrics.Latencies.Max,
					metrics.Latencies.Min,
				), text.WriteReplace())

			d.widgets.bytesText.Write(
				fmt.Sprintf(bytesTextFormat,
					metrics.BytesIn.Total,
					metrics.BytesIn.Mean,
					metrics.BytesOut.Total,
					metrics.BytesOut.Mean,
				), text.WriteReplace())

			d.widgets.othersText.Write(fmt.Sprintf(othersTextFormat,
				metrics.Duration,
				metrics.Wait,
				metrics.Requests,
				metrics.Rate,
				metrics.Throughput,
				metrics.Success,
				metrics.Earliest,
				metrics.Latest,
				metrics.End,
			), text.WriteReplace())

			codesText := ""
			for code, n := range metrics.StatusCodes {
				codesText += fmt.Sprintf(`%q: %d
`, code, n)
			}
			d.widgets.statusCodesText.Write(codesText, text.WriteReplace())

			errorsText := ""
			for i, e := range metrics.Errors {
				errorsText += fmt.Sprintf(`%d: %s
`, i, e)
			}
			d.widgets.errorsText.Write(errorsText, text.WriteReplace())
		}
	}
}
