// Package scribeconf provides utilities for parsing Scribe configuration files.
// For details of Scribe configuration, please refer to
// https://github.com/facebookarchive/scribe/wiki/Scribe-Configuration
package scribeconf

import (
	"fmt"
	"strings"
)

// Store represents a scribe store.
// A store has multiple fields, and can contain multiple sub-stores.
type Store struct {
	Name   string            // Store name denoted by surrounding '<' and '>'
	Fields map[string]string // fields of a store, for example "category"
	Stores []Store           // a list of sub-stores
}

func newStore() *Store {
	s := Store{
		Fields: make(map[string]string),
		Stores: make([]Store, 0),
	}
	return &s
}

// Parse parses a Scribe configuration string as a top level store,
// with possibly multiple sub-stores.
func Parse(input string) (*Store, error) {
	l := lex(input)
	s := newStore()
	if err := parse(s, l.items); err != nil {
		return nil, err
	}
	return s, nil
}

func parse(s *Store, items chan item) error {
	for i := range items {
		switch i.typ {
		case itemError:
			return fmt.Errorf(i.val)
		case itemKey:
			if eq := <-items; eq.typ != itemEqual {
				return fmt.Errorf("%v not followed by eq, but by %v", i, eq)
			}
			val := <-items
			if val.typ != itemVal {
				return fmt.Errorf("%v not followed by a value, but by %v", i, val)
			}
			s.Fields[strings.TrimSpace(i.val)] = strings.TrimSpace(val.val)
		case itemLeftMeta:
			ss := newStore()
			ss.Name = i.val[1 : len(i.val)-1] // extract name from <...>
			err := parse(ss, items)
			if err != nil {
				return err
			}
			s.Stores = append(s.Stores, *ss)
		case itemRightMeta:
			closeName := i.val[2 : len(i.val)-1] // extract name from </...>
			if closeName != s.Name {
				return fmt.Errorf("store %v closed with </%s>", s, closeName)
			}
			return nil
		case itemEOF:
			return nil
		}
	}

	return fmt.Errorf("unexpected EOF for store: %v", s)
}
