package scribeconf

import (
	"testing"
)

func TestParse(t *testing.T) {
	input := `
##  Copyright (c) 2007-2008 Facebook
##
##  Licensed under the Apache License, Version 2.0 (the "License");
##  you may not use this file except in compliance with the License.
##  You may obtain a copy of the License at
##
##      http://www.apache.org/licenses/LICENSE-2.0
##
##  Unless required by applicable law or agreed to in writing, software
##  distributed under the License is distributed on an "AS IS" BASIS,
##  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
##  See the License for the specific language governing permissions and
##  limitations under the License.
##
## See accompanying file LICENSE or visit the Scribe site at:
## http://developers.facebook.com/scribe/


##
## Sample Scribe configuration
##

# This file configures Scribe to listen for messages on port 1463 and write
# them to /tmp/scribetest
#
# This configuration also tells Scribe to discard messages with a category
# that begins with 'ignore'.
#
# If the message category is 'bucket_me', Scribe will hash this message to
# 1 of 5 buckets.

port=1463
max_msg_per_second=2000000
check_interval=3

# IGNORE* - discards messages for categories that begin with 'ignore'
<store>
category=ignore*
type=null
</store>


# BUCKET_ME - write 'bucket_me' messages to 1 of 5 subdirectories
<store>
category=bucket_me
type=buffer

target_write_size=20480
max_write_interval=1
buffer_send_rate=2
retry_interval=30
retry_interval_range=10

<primary>
type=bucket
num_buckets=5
bucket_subdir=bucket
bucket_type=key_hash
delimiter=58
# This will hash based on the part of the message before the first ':' (char(58))

<bucket>
type=file
fs_type=std
file_path=/tmp/scribetest
base_filename=bucket_me
max_size=10000
</bucket>
</primary>

<secondary>
type=file
fs_type=std
file_path=/tmp
base_filename=bucket_me
max_size=30000
</secondary>
</store>


# DEFAULT - write all other categories to /tmp/scribetest
<store>
category=default
type=buffer

target_write_size=20480
max_write_interval=1
buffer_send_rate=2
retry_interval=30
retry_interval_range=10

<primary>
type=file
fs_type=std
file_path=/tmp/scribetest
base_filename=thisisoverwritten
max_size=1000000
</primary>

<secondary>
type=file
fs_type=std
file_path=/tmp
base_filename=thisisoverwritten
max_size=3000000
</secondary>
</store>
  `
	s, err := Parse(input)
	if err != nil {
		t.Fatalf("%v", err)
	}

	check(t, *s, "port", "1463")
	check(t, *s, "max_msg_per_second", "2000000")
	check(t, *s, "check_interval", "3")

	ignore := s.Stores[0]
	check(t, ignore, "category", "ignore*")
	check(t, ignore, "type", "null")

	bucket_me := s.Stores[1]
	check(t, bucket_me, "category", "bucket_me")
	check(t, bucket_me, "type", "buffer")
	check(t, bucket_me, "target_write_size", "20480")
	check(t, bucket_me, "max_write_interval", "1")
	check(t, bucket_me, "retry_interval", "30")
	check(t, bucket_me, "retry_interval_range", "10")
	pm := bucket_me.Stores[0]
	check(t, pm, "type", "bucket")
	check(t, pm, "num_buckets", "5")
	check(t, pm, "bucket_subdir", "bucket")
	check(t, pm, "bucket_type", "key_hash")
	check(t, pm, "delimiter", "58")
	bc := pm.Stores[0]
	check(t, bc, "type", "file")
	check(t, bc, "fs_type", "std")
	check(t, bc, "file_path", "/tmp/scribetest")
	check(t, bc, "base_filename", "bucket_me")
	check(t, bc, "max_size", "10000")
	sc := bucket_me.Stores[1]
	check(t, sc, "type", "file")
	check(t, sc, "fs_type", "std")
	check(t, sc, "file_path", "/tmp")
	check(t, sc, "base_filename", "bucket_me")
	check(t, sc, "max_size", "30000")

	defaul := s.Stores[2]
	check(t, defaul, "category", "default")
	check(t, defaul, "type", "buffer")
	check(t, defaul, "target_write_size", "20480")
	check(t, defaul, "max_write_interval", "1")
	check(t, defaul, "buffer_send_rate", "2")
	check(t, defaul, "retry_interval", "30")
	check(t, defaul, "retry_interval_range", "10")
	pm = defaul.Stores[0]
	check(t, pm, "type", "file")
	check(t, pm, "fs_type", "std")
	check(t, pm, "file_path", "/tmp/scribetest")
	check(t, pm, "base_filename", "thisisoverwritten")
	check(t, pm, "max_size", "1000000")
	sc = defaul.Stores[1]
	check(t, sc, "type", "file")
	check(t, sc, "fs_type", "std")
	check(t, sc, "file_path", "/tmp")
	check(t, sc, "base_filename", "thisisoverwritten")
	check(t, sc, "max_size", "3000000")
}

func check(t *testing.T, s Store, key, value string) {
	if s.Fields[key] != value {
		t.Errorf("key %s does not have value %s in %v", key, value, s)
	}
}
