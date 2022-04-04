package lardoon

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/b1naryth1ef/jambon"
	"github.com/b1naryth1ef/jambon/tacview"
)

var nonHumanSlotRe = regexp.MustCompile(`([A-Z\d]+)#\d\d\d-\d\d`)

type objectData struct {
	Id            uint64
	Types         string
	Name          string
	Pilot         string
	CreatedOffset int
	DeletedOffset int
}

func ImportPath(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.Size() == 0 {
				log.Printf("Skipping empty file %v", path)
				return nil
			}

			if strings.HasSuffix(path, ".acmi") {
				err := ImportFile(path)
				if err != nil {
					log.Printf("Failed to process path %v: %v", path, err)
				}
			}

			return nil
		})
	}

	return ImportFile(path)
}

func ImportFile(target string) error {
	var err error
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}

	stat, err := os.Stat(target)
	if err != nil {
		return err
	}

	file, err := jambon.OpenReadableTacView(target)
	if err != nil {
		return err
	}

	reader, err := tacview.NewReader(file)
	if err != nil {
		return err
	}

	rootObject := reader.Header.InitialTimeFrame.Get(0)

	replayId, err := createReplay(
		target,
		reader.Header.ReferenceTime.String(),
		rootObject.Get("RecordingTime").Value,
		rootObject.Get("Title").Value,
		rootObject.Get("DataSource").Value,
		rootObject.Get("DataRecorder").Value,
		int(stat.Size()),
	)
	if err != nil {
		return err
	}

	if replayId == -1 {
		return nil
	}

	log.Printf("Importing replay %v (#%v)", target, replayId)

	timeFrames := make(chan *tacview.TimeFrame)

	objects := make(map[uint64]*objectData)
	done := make(chan struct{})
	var lastFrame int
	var firstFrame int

	go func() {
		defer close(done)

		for {
			tf, ok := <-timeFrames
			if !ok {
				return
			}
			if int(tf.Offset) > 0 && int(tf.Offset) < firstFrame {
				firstFrame = int(tf.Offset)
			}
			if int(tf.Offset) > lastFrame {
				lastFrame = int(tf.Offset)
			}

			for _, object := range tf.Objects {
				_, exists := objects[object.Id]
				if object.Deleted && exists {
					objects[object.Id].DeletedOffset = int(tf.Offset)
					err := createReplayObject(
						replayId,
						int(object.Id),
						objects[object.Id].Types,
						objects[object.Id].Name,
						objects[object.Id].Pilot,
						objects[object.Id].CreatedOffset,
						objects[object.Id].DeletedOffset,
					)
					if err != nil {
						panic(err)
					}
					delete(objects, object.Id)
				} else if !exists {
					types := object.Get("Type")
					if types != nil {
						if strings.Contains(types.Value, "Air") && (strings.Contains(types.Value, "FixedWing") || strings.Contains(types.Value, "Rotorcraft")) {
							name := object.Get("Name").Value

							pilotProp := object.Get("Pilot")
							if pilotProp == nil {
								continue
							}
							pilot := pilotProp.Value
							group := object.Get("Group").Value

							result := nonHumanSlotRe.FindAllStringSubmatch(pilot, -1)
							if len(result) > 0 && strings.HasPrefix(group, result[0][1]) {
								continue
							}

							objects[object.Id] = &objectData{
								Id:            object.Id,
								Name:          name,
								Pilot:         pilot,
								Types:         types.Value,
								CreatedOffset: int(tf.Offset),
							}
						}
					}
				}
			}

		}
	}()

	err = reader.ProcessTimeFrames(runtime.GOMAXPROCS(-1), timeFrames)
	<-done
	for _, object := range objects {
		err := createReplayObject(
			replayId,
			int(object.Id),
			object.Types,
			object.Name,
			object.Pilot,
			object.CreatedOffset,
			lastFrame,
		)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	return setReplayDuration(replayId, lastFrame-firstFrame)
}
