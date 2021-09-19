package mt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

func (tc ToolCaps) String() string {
	b, err := tc.MarshalJSON()
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (tc ToolCaps) MarshalJSON() ([]byte, error) {
	if !tc.NonNil {
		return []byte("null"), nil
	}

	var dgs bytes.Buffer
	dgs.WriteByte('{')
	e := json.NewEncoder(&dgs)
	for i, dg := range tc.DmgGroups {
		e.Encode(dg.Name)
		dgs.WriteByte(':')
		e.Encode(dg.Rating)
		if i < len(tc.DmgGroups)-1 {
			dgs.WriteByte(',')
		}
	}
	dgs.WriteByte('}')

	var gcs bytes.Buffer
	gcs.WriteByte('{')
	e = json.NewEncoder(&gcs)
	for i, gc := range tc.GroupCaps {
		var maxRating int16
		for _, t := range gc.Times {
			if t.Rating >= maxRating {
				maxRating = t.Rating + 1
			}
		}

		times := make([]interface{}, maxRating)
		for _, t := range gc.Times {
			times[t.Rating] = fmtFloat(t.Time)
		}

		e.Encode(gc.Name)
		gcs.WriteByte(':')
		e.Encode(map[string]interface{}{
			"uses":     gc.Uses,
			"maxlevel": gc.MaxLvl,
			"times":    times,
		})
		if i < len(tc.GroupCaps)-1 {
			gcs.WriteByte(',')
		}
	}
	gcs.WriteByte('}')

	return json.Marshal(map[string]interface{}{
		"damage_groups":       json.RawMessage(dgs.Bytes()),
		"full_punch_interval": fmtFloat(tc.AttackCooldown),
		"groupcaps":           json.RawMessage(gcs.Bytes()),
		"max_drop_level":      tc.MaxDropLvl,
		"punch_attack_uses":   tc.PunchUses,
	})
}

func (tc *ToolCaps) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))

	t, err := d.Token()
	if err != nil {
		return err
	}
	if t == nil {
		*tc = ToolCaps{}
		return nil
	}
	if d, ok := t.(json.Delim); !ok || d != '{' {
		return errors.New("not an object")
	}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return err
		}
		key := t.(string)

		err = nil
		switch key {
		case "full_punch_interval":
			err = d.Decode(&tc.AttackCooldown)
		case "max_drop_level":
			err = d.Decode(&tc.MaxDropLvl)
		case "groupcaps":
			tc.GroupCaps = nil

			t, err := d.Token()
			if err != nil {
				return fmt.Errorf("groupcaps: %w", err)
			}
			if d, ok := t.(json.Delim); !ok || d != '{' {
				return errors.New("groupcaps: not an object")
			}
			for d.More() {
				var gc ToolGroupCap

				t, err := d.Token()
				if err != nil {
					return fmt.Errorf("groupcaps: %w", err)
				}
				gc.Name = t.(string)

				t, err = d.Token()
				if err != nil {
					return fmt.Errorf("groupcaps: %w", err)
				}
				if d, ok := t.(json.Delim); !ok || d != '{' {
					return errors.New("groupcaps: not an object")
				}
				for d.More() {
					t, err := d.Token()
					if err != nil {
						return fmt.Errorf("groupcaps: %w", err)
					}
					key := t.(string)

					err = nil
					switch key {
					case "uses":
						err = d.Decode(&gc.Uses)
					case "maxlevel":
						err = d.Decode(&gc.MaxLvl)
					case "times":
						gc.Times = nil

						t, err := d.Token()
						if err != nil {
							return fmt.Errorf("groupcaps: times: %w", err)
						}
						if d, ok := t.(json.Delim); !ok || d != '[' {
							return errors.New("groupcaps: times: not an array")
						}
						for i := int16(0); d.More(); i++ {
							t, err := d.Token()
							if err != nil {
								return fmt.Errorf("groupcaps: times: %w", err)
							}
							switch t := t.(type) {
							case nil:
							case float64:
								gc.Times = append(gc.Times, DigTime{i, float32(t)})
							default:
								return errors.New("groupcaps: times: not null or a number")
							}
						}
						_, err = d.Token()
					}
					if err != nil {
						return fmt.Errorf("groupcaps: %s: %w", key, err)
					}
				}
				if _, err := d.Token(); err != nil {
					return err
				}
				tc.GroupCaps = append(tc.GroupCaps, gc)
			}
			_, err = d.Token()
		case "damage_groups":
			tc.DmgGroups = nil

			t, err := d.Token()
			if err != nil {
				return fmt.Errorf("damage_groups: %w", err)
			}
			if d, ok := t.(json.Delim); !ok || d != '{' {
				return errors.New("damage_groups: not an object")
			}
			for d.More() {
				var g Group

				t, err := d.Token()
				if err != nil {
					return err
				}
				g.Name = t.(string)

				if err := d.Decode(&g.Rating); err != nil {
					return fmt.Errorf("damage_groups: %w", err)
				}

				tc.DmgGroups = append(tc.DmgGroups, g)
			}
			_, err = d.Token()
		case "punch_attack_uses":
			err = d.Decode(&tc.PunchUses)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	if _, err := d.Token(); err != nil {
		return err
	}

	tc.NonNil = true
	return nil
}

func fmtFloat(f float32) json.Number {
	buf := make([]byte, 0, 24)
	buf = strconv.AppendFloat(buf, float64(f), 'g', 17, 32)
	if !bytes.ContainsRune(buf, '.') {
		buf = append(buf, '.', '0')
	}
	return json.Number(buf)
}
