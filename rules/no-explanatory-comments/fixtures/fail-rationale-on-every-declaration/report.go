package report

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Record is one file's verdict. The elapsed time sits beside the verdict
// rather than on the summary, because a slow file is a property of the file,
// and a summary holding a map of durations would be a second copy of the
// file list.
type Record struct {
	File    string
	Passed  bool
	Elapsed time.Duration
}

// writer is one output format at both its moments: what it writes as records
// arrive and what it writes once the run is over. Holding both on one struct
// keeps the choice of format a table lookup rather than a branch at each
// moment.
type writer struct {
	record func(Record) error
	close  func(int) error
}

// writers maps the flag to its format. A map rather than a switch, so adding
// a format is one entry here rather than a new case in every place the
// choice appears.
var writers = map[string]func(io.Writer) writer{
	"text": textWriter,
	"json": jsonWriter,
}

// textWriter streams each record as it lands. Streaming is the point: a run
// over a large corpus takes minutes, and a report that arrives only at the
// end reads as a hang from the outside.
func textWriter(w io.Writer) writer {
	return writer{
		record: func(r Record) error {
			_, err := fmt.Fprintf(w, "%s %v %s\n", r.File, r.Passed, r.Elapsed)
			return err
		},
		close: func(failed int) error {
			_, err := fmt.Fprintf(w, "%d failed\n", failed)
			return err
		},
	}
}

// jsonWriter holds everything until the run is over. One envelope covering
// every record is a single document, and half a document is not parseable,
// so buffering is what makes the format worth offering at all.
func jsonWriter(w io.Writer) writer {
	var records []Record
	return writer{
		record: func(r Record) error {
			records = append(records, r)
			return nil
		},
		close: func(int) error {
			return json.NewEncoder(w).Encode(records)
		},
	}
}
