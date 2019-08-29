// Code generated by vfsgen; DO NOT EDIT.

// +build !dev

package config

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	pathpkg "path"
	"time"
)

// assets statically implements the virtual filesystem provided to vfsgen.
var assets = func() http.FileSystem {
	fs := vfsgen۰FS{
		"/": &vfsgen۰DirInfo{
			name:    "/",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 493622726, time.UTC),
		},
		"/orchestrations": &vfsgen۰DirInfo{
			name:    "orchestrations",
			modTime: time.Date(2019, 9, 12, 18, 21, 13, 734228583, time.UTC),
		},
		"/orchestrations/.DS_Store": &vfsgen۰CompressedFileInfo{
			name:             ".DS_Store",
			modTime:          time.Date(2019, 9, 12, 18, 21, 18, 737828864, time.UTC),
			uncompressedSize: 6148,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x98\x41\x4e\xf2\x40\x18\x86\xdf\x99\x9f\x84\xfe\xe2\xa2\x4b\x97\xbd\x00\x0b\x6e\x50\xb1\x9e\x80\xa5\x1b\x0b\x45\x24\x96\x4e\xd3\x6a\x13\x77\xbd\x82\x31\x9e\xc1\x73\x78\x11\xcf\x62\xda\x79\x4d\x6a\x29\x04\x16\xc6\x6a\xbe\x27\x21\x4f\x02\x6f\x87\x6f\x3a\x64\xf8\xa6\x00\xd4\xf4\x21\x9a\x00\x2e\x00\x07\xd6\xfa\x3f\x3a\x71\xf8\xda\x42\xd3\xa3\x6a\xbc\x7a\x0c\x83\x08\x8f\x18\xc3\x20\xed\x1e\x4b\xe8\x19\xa3\x7a\xf1\x43\xa4\x48\x91\x7f\x59\xbf\x25\x32\x84\xb8\x87\x41\x56\xcc\xb2\x24\x36\xc9\xca\xae\x33\x4e\xb1\x40\x8c\x35\xc6\xc8\xeb\x54\x81\x35\x16\x58\x22\x8f\x57\x93\xd9\xc2\x6c\x9a\x6b\xbf\x33\xbb\x31\x41\x30\x8f\xcd\xbc\xfa\x71\x3d\xbd\x5f\x8d\xcd\xf3\xdb\xf9\xfe\x7c\x74\x54\x3e\xbd\x3d\xbc\x96\xd6\xfc\x86\xb8\xe3\x1d\x49\x10\xa2\xe8\x98\x55\x2b\xd1\x9c\xcb\xcd\xeb\xc5\xc9\xfc\xa5\xae\x6d\x2b\x15\x1d\x90\xea\xa8\x7b\x88\x18\x21\x12\x44\x58\x23\xc1\xaa\x55\xad\x20\x08\xc2\xf1\x70\xf7\x70\x46\x3f\x5d\x88\x20\x08\xbd\xa3\xda\x1f\x3c\xda\xa7\x4b\x6b\xc5\xcf\x35\x3d\x68\x5c\xe3\xd2\x1e\xed\xd3\xa5\xb5\x62\x4e\xd3\x03\xda\xa1\x5d\xda\xa3\x7d\xba\xb4\xe6\xa6\xa5\x78\xf8\x50\xfc\x66\xc5\x13\x8a\x72\x69\x8f\xf6\xbf\xe7\xde\x08\xc2\x6f\xe7\x9f\x95\x5b\xfd\xff\x5f\xee\x3e\xff\x0b\x82\xf0\x87\x51\x83\x60\x16\x4c\xf7\x3c\x4e\xd0\x6c\x04\xae\x3f\x2f\x68\x35\x02\x68\x34\x01\xda\x3e\x2c\x3c\x6b\xbc\x2f\x8d\x80\x20\xf4\x8c\x8f\x00\x00\x00\xff\xff\x78\x96\x7b\x05\x04\x18\x00\x00"),
		},
		"/orchestrations/appsody-operator": &vfsgen۰DirInfo{
			name:    "appsody-operator",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 488559236, time.UTC),
		},
		"/orchestrations/appsody-operator/.DS_Store": &vfsgen۰CompressedFileInfo{
			name:             ".DS_Store",
			modTime:          time.Date(2019, 9, 9, 19, 24, 11, 127780520, time.UTC),
			uncompressedSize: 6148,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\xd8\x31\x0a\x02\x31\x10\x85\xe1\x37\x31\x45\xc0\x26\xa5\x65\x1a\x0f\xe0\x0d\xc2\xb2\x9e\xc0\x0b\x58\x78\x05\xfb\x1c\x5d\x96\x79\x60\x60\xd5\x4e\x8c\xcb\xfb\x40\xfe\x05\x37\x2a\x16\x31\x23\x00\x9b\xee\xb7\x13\x90\x01\x24\x78\x71\xc4\x4b\x89\x8f\x95\xd0\x5d\x1b\x5f\x43\x44\x44\x44\xc6\x66\x9e\xb4\xff\xf5\x07\x11\x91\xe1\x2c\xfb\x43\x61\x2b\xdb\xbc\xc6\xe7\x03\x1b\xbb\x35\x99\x2d\x6c\x65\x9b\xd7\x78\x5f\x60\x23\x9b\xd8\xcc\x16\xb6\xb2\xcd\xcb\x4d\xcb\x38\x7c\x18\xdf\xd9\x38\xa1\x18\xa7\x10\x2b\x6c\xfd\xce\x77\x23\xf2\xef\x76\x9e\xbc\xfc\xfe\x9f\xdf\xcf\xff\x22\xb2\x61\x16\xe7\xcb\x3c\x3d\x07\x82\xf5\x0d\x00\xae\xdd\xf5\xa7\x43\x40\xf0\x3f\x0b\x0f\xdd\x5a\x1d\x04\x44\x06\xf3\x08\x00\x00\xff\xff\x6a\x00\x88\x6d\x04\x18\x00\x00"),
		},
		"/orchestrations/appsody-operator/0.1": &vfsgen۰DirInfo{
			name:    "0.1",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 488682784, time.UTC),
		},
		"/orchestrations/appsody-operator/0.1/appsody.yaml": &vfsgen۰CompressedFileInfo{
			name:             "appsody.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 489269279, time.UTC),
			uncompressedSize: 7504,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x59\x5f\x8f\xdb\xb8\x11\x7f\xf7\xa7\x18\xa4\x0f\xdb\x16\xb5\x16\x69\x70\x87\xc2\xc0\xa1\x75\x9d\x5c\x9b\xde\x25\x6b\xd8\x9b\x2b\x8a\xa2\x38\x8c\xa9\xb1\xcd\x2e\x45\xaa\xe4\xd0\x17\x27\xc8\x77\x2f\x48\x4a\x96\x65\x49\xfe\x93\x4b\xf8\x64\x93\x9c\x99\x1f\xe7\x1f\x87\x23\x2c\xe5\x4f\x64\x9d\x34\x7a\x02\x58\x4a\x7a\xcf\xa4\xc3\x3f\x97\x3d\xfd\xc9\x65\xd2\xdc\xef\x9e\xaf\x88\xf1\xf9\xe8\x49\xea\x7c\x02\x33\xef\xd8\x14\x0b\x72\xc6\x5b\x41\x2f\x69\x2d\xb5\x64\x69\xf4\xa8\x20\xc6\x1c\x19\x27\x23\x00\x8d\x05\x05\x6e\xa5\x33\xf9\x1e\xcb\x52\x49\x81\x1c\x79\x56\x73\x59\x4e\xbb\x91\x2b\x49\x84\xdd\x98\xe7\x91\x05\xaa\xb9\x95\x9a\xc9\xce\x8c\xf2\x85\x76\x61\x6d\x0c\xff\x58\x3e\xbc\x9d\x23\x6f\x27\x90\x05\x82\xec\x88\xdd\xeb\x02\x37\x34\x02\x00\xc8\xc9\x09\x2b\x4b\x8e\xa7\x98\xae\x9c\x51\x9e\x29\xc2\x00\xb3\x06\xde\x12\xe4\x54\x2a\xb3\xa7\x1c\x64\x20\x02\x61\x34\xa3\xd4\x52\x6f\xc0\xd2\x46\x3a\xb6\x7b\x40\x9d\x03\xe3\x26\x32\x4c\x27\x68\x04\xf0\xbe\xa4\x09\x38\xb6\x52\x6f\x7a\x61\xd1\xfb\xd2\xb8\x1e\x30\xcb\x92\x84\x5c\x4b\x72\xf0\xcb\x96\x78\x4b\xb6\x42\x52\x90\x66\x90\x0e\x12\x5d\x0e\x41\xef\x56\xa3\x52\x7b\xd8\x49\x84\x9c\xd6\xe8\x15\xc3\xc2\x78\xa6\x23\x44\xaf\xd2\xf6\x23\x4c\x2b\x63\x14\xa1\xee\x80\x62\x64\xef\x32\x61\x74\x52\xae\xfb\xf7\x9f\x7f\xfb\x97\x2c\x90\x7c\xf7\xdd\xdd\x82\x84\xd1\x42\x2a\xca\xef\x7e\xf7\x9f\x6a\x6b\x0f\xf4\x38\x5f\x2b\xd0\xd6\x34\x70\xe0\x79\x84\xab\xe1\x78\x59\x5d\xd7\x23\xb3\x84\xae\x12\xd3\x42\xb6\x88\xf3\xb0\x36\x36\x42\x5b\xa3\x54\xde\x46\x53\x5f\x42\x79\xe0\x57\x5a\x69\xac\xe4\xfd\x04\x9e\x7f\x49\xc4\x05\x39\xd7\xeb\x94\xdf\x57\x18\xab\x0d\xb0\xb6\xa6\xb8\x80\xf6\xcd\x11\xaf\x5b\xe0\xd6\x91\x98\x09\x4b\x31\x4e\x1e\x65\x41\x8e\xb1\x28\x7b\x62\x65\x43\x8d\x81\x53\x4c\x1f\x21\x98\xb6\xbc\x3f\xc7\xe8\x8a\x1b\x6b\x7c\x79\x88\xee\x18\xc9\x69\x7f\x0c\x58\x80\x94\x27\xa6\x69\x79\xda\x44\x6b\x5c\x54\xd2\xf1\x0f\x03\x1b\x7e\x94\x8e\xd3\x61\x95\xb7\xa8\x7a\x13\x48\x5c\x77\x52\x6f\xbc\x42\xdb\xb7\x63\x04\xe0\x84\x09\x70\xdf\x06\x48\x25\x8a\xe8\x92\xce\xaf\xea\xf3\x55\x30\x93\x51\x27\xf0\xf1\xd3\x08\x60\x87\x4a\xe6\x91\x3e\x2d\x9a\x92\xf4\x74\xfe\xfa\xa7\x17\x4b\xb1\xa5\x02\xd3\x64\xb0\x82\x29\xc9\xb2\xac\x79\x84\x71\x94\x3f\x0f\x73\x27\x4a\xbe\x0b\xac\xd2\x9e\x10\xd8\x52\x93\x8b\x1a\xdf\xa5\x39\xca\xc1\x45\x31\xc9\x12\xd2\x81\xa5\xd2\x92\x23\xcd\x8d\xe2\xea\x61\xd6\x80\x1a\xcc\xea\xbf\x24\x38\x83\x25\xd9\xc0\x04\xdc\xd6\x78\x95\x07\x2f\xda\x91\xe5\xe8\x57\x1b\x2d\x3f\x1c\x38\x3b\x60\x13\x45\x2a\x64\xaa\xb4\x5c\x8f\x98\x72\x35\xaa\xa0\x04\x4f\x7f\x88\x69\xb0\xc0\x3d\x58\x0a\x32\xc0\xeb\x23\x6e\x71\x8b\xcb\xe0\x8d\xb1\x04\x52\xaf\xcd\x04\xb6\xcc\xa5\x9b\xdc\xdf\x6f\x24\xd7\x37\x86\x30\x45\xe1\xb5\xe4\xfd\x7d\x48\xb3\x56\xae\x3c\x1b\xeb\xee\x73\xda\x91\xba\xc7\x52\x8e\x23\x4e\x9d\x6e\x84\x22\xff\xcd\xc1\x32\x77\x47\xc0\x4e\x3c\x3c\x8d\xe8\x5b\x83\x6a\x0e\x8e\x15\x72\x2a\x56\x64\x09\x6e\xa3\xcd\x30\x15\x94\xb0\x78\xb5\x7c\x3c\xb8\x7b\xd4\x78\x5b\xc5\x51\xb9\x0d\x99\x6b\xf4\x1c\xf4\x22\xf5\x9a\x6c\xb2\x53\x0c\xe2\xc0\x91\x74\x5e\x1a\xa9\x39\xfe\x11\x4a\x92\x6e\xeb\xd8\xf9\x55\x21\x39\x18\xf6\x7f\x9e\x1c\x07\x73\x64\x30\x43\xad\x0d\xc3\x8a\xc0\x97\x21\xb6\xf2\x0c\x5e\x6b\x98\x61\x41\x6a\x86\x8e\xbe\xb4\x96\x83\x42\xdd\x38\x68\xf0\xb2\x9e\x8f\x2f\xf3\xf6\xc6\xa4\x9c\xc3\x74\x7d\x87\xd7\xa3\x2f\x42\x52\x94\xb4\x6f\xed\xf6\xea\x00\x8a\x48\x68\xc5\x56\x32\x09\xf6\xb6\x43\x24\x99\x0a\x77\x3a\x79\x86\x57\xbd\x84\xd6\xe2\xbe\x2d\xc5\xb3\x71\x02\x95\xd4\x9b\x53\x7e\x43\x27\x8a\x6a\xc2\xf7\x0b\x8a\x07\xeb\x59\x84\x70\x3f\x15\xc8\x93\x10\x61\x2f\xfe\xd8\xb3\x5e\x48\x2d\x0b\x5f\xd4\x39\xbd\x0f\x6a\x08\xce\x0d\xd9\xae\x64\xa9\x7f\x8d\xe4\xf3\xcc\x19\xed\x86\x78\x36\x7f\xf7\x8e\xa5\x92\x1f\xa2\xd5\xe6\x64\x45\x48\x48\x5d\xd3\xfd\x5a\x79\xbd\x7e\x15\x46\xbc\xbe\xe8\x07\x8d\x2c\x77\x14\x22\x50\x8a\x01\xbf\x69\x6a\xa0\x66\x90\xde\xdd\xe4\x2f\x3d\x00\x86\xfd\x85\xf4\xee\x7b\x6b\x8a\xaf\x28\x20\xd6\x79\xd7\x9f\x56\xc9\x1d\x69\x72\x6e\x6e\xcd\x6a\x80\xac\x47\x7e\xe9\x95\x9a\x1b\x25\xc5\xfe\xea\x78\x0c\x24\x4b\x12\x96\xf8\x6a\x12\x4b\x98\xcb\xdb\xc1\xd9\x01\xff\x3e\xe7\x6b\xc3\x7e\x56\xa7\xfb\x99\xd1\x8e\x2d\x4a\xcd\x1d\xc6\x83\x48\x5c\xbf\xef\x9d\xcb\x0c\x31\xb5\xa7\x92\xa5\x2f\x5e\x8e\x9f\x3c\xc3\x4c\xe0\x7c\x32\x3b\x8b\x39\x02\x34\xb6\x63\x25\xb8\x26\x2f\xe1\xfb\x94\x97\xbe\xfd\xe6\x9b\x17\xdf\x7e\xd9\xc4\x15\x57\x07\xa9\xce\x24\xed\x61\xbb\x4c\x85\x30\x5e\x73\xa8\xf7\xae\xf6\x49\xc7\x28\x9e\x6e\xd8\x6d\x6c\x4f\xe6\x3b\x7b\x35\x04\x48\xb1\x20\xbf\xed\xb0\xa1\xba\xfd\x70\xab\x86\x00\x76\xe1\xc9\x4c\x33\x85\xb2\x78\xa4\xa2\x0c\x65\xde\x30\x8f\x33\x99\xa8\x67\x29\xb1\x7e\x13\xce\xd3\x39\xe7\x97\x4b\x78\x49\xca\x57\x13\x10\x0a\x2f\x69\xa9\x55\x3c\x8e\x3b\x55\x49\x6b\x31\xba\xc8\xc5\xfa\x27\xbd\x22\xae\xa8\x80\x9a\xd7\xe3\x95\x67\x3c\xe7\x5d\x00\x0a\x1d\x3f\x5a\xd4\x4e\xd6\x2f\xbc\xfe\xfc\x51\xc7\x7a\x28\x30\xc7\x2c\x0b\xfa\x9c\x2c\x13\x84\xbd\x8b\x35\xea\x57\x16\x54\xbd\x8e\x3f\x2b\x15\xa6\x66\xc1\x67\x91\x76\xad\x78\x35\xe9\x50\x3e\xbb\x40\x78\xb3\x03\x9f\x10\xec\xea\x96\x5d\xdd\x9d\x3b\x4c\x55\x6d\xb3\xf4\x80\x6f\x56\x53\xb6\xa4\x7c\x02\x6c\x3d\x55\x2f\xe0\x94\xd5\xd2\xcc\x78\x3c\x1e\x1d\xf7\x02\x77\x75\xc7\x6f\xd9\xca\xb2\xc3\x7d\xbe\x71\x70\x57\x64\x63\x3b\xac\xec\x0a\x45\x86\x9e\xb7\xc6\x56\xd5\x64\xd3\x5b\xac\xdb\x8a\xca\x3b\x26\xbb\x30\x8a\x5a\x12\x3a\x2d\x8c\x09\x68\xaf\xd4\xb0\x6c\xeb\x55\x08\x98\x10\xdb\xf2\x6f\xd6\xf8\xb2\xd2\xc7\xb3\x67\xa3\xa6\x08\xa8\xe6\x4a\x93\xbb\xf8\xa3\xba\x48\xd2\x9f\xfa\x2d\x97\xfe\x95\xe1\x10\x8e\x49\x73\xca\x50\x22\xa4\xd8\x6a\x63\x78\x5d\xa5\x9f\xc2\xe8\xb5\xdc\x14\x58\xd6\xfc\x42\x81\xd4\xe2\x8d\x49\x7d\x2e\x19\x6a\x55\x21\xb8\xfb\xfd\x5d\x17\x6a\x38\x53\x17\x6c\xd3\x36\x4c\x7c\x73\xa4\xc2\x68\x57\x8b\xa9\xab\xa5\x83\x58\x46\xa6\xb5\x57\xd5\xc4\x45\x99\xcd\x03\xa8\x2b\x3a\xda\xcd\x68\x46\x55\x9a\xbc\xde\x49\xf6\x1a\xbe\x85\xd1\x92\x4d\x08\x81\x4c\x18\x4b\xc6\x65\xc2\x14\x5d\x09\x95\x92\xaa\xdd\x27\x8c\x37\xc4\x49\xc9\xf1\x3d\x70\x51\x5f\x6f\xeb\x56\xd4\xb8\xeb\x1d\xe7\xb4\x7a\xbf\x96\x1a\x95\xfc\xd0\x39\x58\x7a\x9d\xf7\xcb\x6d\x3a\x60\x27\x8c\x83\x36\x2e\xab\xc7\x1a\xcf\x94\x99\x92\xb4\xdb\xca\x35\x67\xd2\x8c\x00\x90\xd3\x6b\x9e\x16\x14\xb2\x87\x48\x97\x46\xed\xf7\x27\x72\x22\x87\x6b\x2c\x11\x55\xac\x37\xd9\x53\x7a\x4f\x55\xa8\x6f\x92\x75\x14\x26\x97\xa4\x3d\xe1\x0a\x35\x59\x93\x4e\x74\xc2\xa7\x5e\xec\xb1\xf3\x67\x32\xba\x68\xbb\xf1\xb8\x9b\x68\xfe\x2a\x75\x1e\xfc\xfd\xea\x54\x75\x45\xea\x73\x3e\x66\xe8\x98\x81\x7a\xd3\xe7\x20\x65\xd5\x43\x2d\x51\xd0\x04\xa6\xf3\xf9\xf2\xe1\xe5\xbf\x7e\x7e\x98\xbf\x5a\x4c\x1f\x1f\x16\x3f\xbf\x9d\xbe\x79\xb5\x9c\x4f\x67\xaf\x60\x64\x8d\xa2\x05\xad\x03\x88\x6e\xee\x3c\xc3\xbe\x56\xec\x99\x43\x06\x3d\x01\x8c\xda\x9f\x84\x4a\xd7\xa4\xe9\x97\x87\x70\xb9\x4a\x19\x55\x73\xe8\xf0\x92\x8b\xef\x05\x47\x8a\x04\x1b\x9b\x6e\xcd\x02\x59\x6c\x7f\xc4\x15\xa9\xc3\xfd\x3b\x78\x02\x6e\x55\xb6\xa7\x3d\x2a\xd5\x62\x72\x86\x4d\xbb\x6b\xd5\xf3\x88\xe8\x27\x82\xfa\xa3\x11\xd9\x23\x29\xe3\x73\x72\xd2\x88\x5f\x9c\x26\xf0\xf1\x23\x64\xe9\xe3\xd3\xa7\x4f\x47\xab\xc2\x14\x05\xea\x6e\x61\x7a\x96\xdb\xbc\x79\xb8\xc3\x54\xfd\x82\xfb\xe3\x0e\x66\xa7\xf3\x51\x63\xfc\xe7\xf4\x71\xf6\xf7\xc6\x95\x4e\xea\x8e\xd8\x2c\x8d\x5d\x8d\x6e\x2d\x13\x47\xa7\x88\x59\x4b\x52\x79\x70\xc5\x2b\xf7\x57\x14\xe9\x33\xc5\xe1\x2b\xc5\xc1\xed\x7b\x31\xcf\x1f\x5e\x46\xc4\x83\x60\x87\x51\xdd\x20\xbf\x57\x74\x2b\xf8\xfa\xe4\x4f\xe0\xd9\xa9\xa1\x9e\x8d\xfe\x1f\x00\x00\xff\xff\xce\xec\xc0\x0c\x50\x1d\x00\x00"),
		},
		"/orchestrations/cli-services": &vfsgen۰DirInfo{
			name:    "cli-services",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 489429489, time.UTC),
		},
		"/orchestrations/cli-services/.DS_Store": &vfsgen۰CompressedFileInfo{
			name:             ".DS_Store",
			modTime:          time.Date(2019, 9, 10, 11, 39, 48, 473324871, time.UTC),
			uncompressedSize: 6148,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\xd8\x31\x0a\x02\x31\x10\x85\xe1\x37\x31\x45\xc0\x26\xa5\x65\x1a\x0f\xe0\x0d\xc2\xb2\x9e\xc0\x0b\x58\x78\x05\xfb\x1c\x5d\x96\x79\x60\x60\xd5\x4e\x8c\xcb\xfb\x40\xfe\x05\x37\x2a\x16\x31\x23\x00\x9b\xee\xb7\x13\x90\x01\x24\x78\x71\xc4\x4b\x89\x8f\x95\xd0\x5d\x1b\x5f\x43\x44\x44\x44\xc6\x66\x9e\xb4\xff\xf5\x07\x11\x91\xe1\x2c\xfb\x43\x61\x2b\xdb\xbc\xc6\xe7\x03\x1b\xbb\x35\x99\x2d\x6c\x65\x9b\xd7\x78\x5f\x60\x23\x9b\xd8\xcc\x16\xb6\xb2\xcd\xcb\x4d\xcb\x38\x7c\x18\xdf\xd9\x38\xa1\x18\xa7\x10\x2b\x6c\xfd\xce\x77\x23\xf2\xef\x76\x9e\xbc\xfc\xfe\x9f\xdf\xcf\xff\x22\xb2\x61\x16\xe7\xcb\x3c\x3d\x07\x82\xf5\x0d\x00\xae\xdd\xf5\xa7\x43\x40\xf0\x3f\x0b\x0f\xdd\x5a\x1d\x04\x44\x06\xf3\x08\x00\x00\xff\xff\x6a\x00\x88\x6d\x04\x18\x00\x00"),
		},
		"/orchestrations/cli-services/0.1": &vfsgen۰DirInfo{
			name:    "0.1",
			modTime: time.Date(2019, 9, 13, 20, 6, 40, 496110958, time.UTC),
		},
		"/orchestrations/cli-services/0.1/kabanero-cli.yaml": &vfsgen۰CompressedFileInfo{
			name:             "kabanero-cli.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 7, 46, 347233287, time.UTC),
			uncompressedSize: 2008,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x54\x4f\x4f\xdb\x4e\x10\xbd\xfb\x53\x8c\xf8\x1d\x90\x7e\x92\x8d\x50\x39\xb4\xbe\x05\x4a\xab\xaa\x40\xa3\x80\x7a\x45\x93\xcd\xc4\xde\xb2\xde\x59\xed\x8e\x83\x00\xf1\xdd\xab\xf5\x9f\xe0\xc4\xa6\xa0\xd6\x97\x38\x3b\xe3\x37\x6f\x76\xde\x3c\x74\xfa\x27\xf9\xa0\xd9\xe6\x80\xce\x85\xa3\xcd\x71\x72\xa7\xed\x2a\x87\xcf\xe4\x0c\x3f\x54\x64\x25\xa9\x48\x70\x85\x82\x79\x02\x60\xb1\xa2\x1c\xee\x70\x89\x96\x3c\xa7\xca\xe8\x24\x38\x52\x31\x14\xc8\x90\x12\xf6\xf1\x1d\xa0\x42\x51\xe5\x05\x2e\xc9\x84\xf6\x00\x62\x81\xbd\x4f\x01\x3c\x39\xa3\x15\x86\x1c\x8e\x13\x00\xa1\xca\x19\x14\xea\x20\x06\x75\xe3\x63\x76\xd0\xa6\xf1\x00\x7a\x3a\xcd\x3b\xf9\x8d\x56\x34\x53\x8a\x6b\x2b\x57\x63\xee\x6d\x9a\x62\x2b\xa8\x2d\xf9\x01\x78\x3a\xd5\x2a\x6c\x9f\xff\x60\x41\xce\xa0\x22\x90\x52\x07\xb8\xd7\x52\x82\x94\x04\xcb\x5a\x1b\x01\x5d\x61\x41\x0d\xc0\xe0\x93\xe6\x30\x87\xa7\x27\xc8\xda\xf8\xf3\xf3\x20\xea\xd8\xcb\xa0\x7c\x24\xb0\xa5\x35\x67\x2f\x39\x7c\x3a\x39\xf9\xb0\x0f\x37\xaf\x8d\x99\xb3\xd1\xea\x21\x87\x99\xb9\xc7\x87\x30\xc8\x20\xbb\x19\x02\xbe\xf4\xf4\x7d\x76\x3a\xbb\x3a\x5f\xfc\xb8\x3d\xbb\xf8\x76\x7b\x35\xbb\x3c\xbf\x9e\xcf\xce\xce\x77\x52\x01\x36\x68\x6a\xfa\xe2\xb9\xca\xf7\x02\x00\x6b\x4d\x66\xb5\xa0\xf5\x38\xd2\xc5\xe6\x28\x65\xbe\x9d\x5f\x16\xab\x06\x87\x8a\x76\xc9\x8d\xc1\x9b\x9e\xd7\xba\xb8\x44\x37\x09\x3f\x1e\x49\xea\xd9\x50\xda\x7e\x95\xa4\x69\x9a\x0c\x15\xbd\x15\xf3\x75\xab\x83\xbf\x53\xf2\x94\xcc\xb6\xd3\x4a\xc1\x79\x16\x56\x6c\x72\xb8\x39\x9b\x27\xfd\x28\x73\xe8\xa7\x25\xe8\x0b\x92\xc1\x08\xf7\x59\x7a\xae\x85\x32\x76\x64\x43\xa9\xd7\x92\x69\x7e\xd9\xc2\x45\x8c\xbd\x97\xb6\x70\x4b\x78\xb7\xe7\xe9\x7b\x8b\xe9\xfd\x2e\x09\xf9\x4a\x5b\x94\x86\x8d\xc3\x10\xa4\xf4\x5c\x17\xe5\x98\xe9\x12\x55\x86\xb5\x94\xec\xf5\x63\x93\x9f\xdd\x7d\x0c\x7b\x84\xcd\x2e\x5f\xe5\xa9\xc9\xbc\xd1\x15\x05\xc1\xca\xe5\x60\x6b\x63\xa6\x3b\xf1\xb5\xa1\x90\x27\x29\xa0\xd3\x5f\x3d\xd7\xae\xbb\xe3\x83\x83\xc6\x2a\x02\xd7\x5e\x51\x7f\xef\xbc\x8a\x62\xdf\x90\x5f\x76\x27\x05\x49\xf3\x6b\x74\x68\x5f\xee\xa3\x07\xbd\x0f\xad\x55\x50\x85\x2e\x34\x7f\x3b\xe3\x78\xa5\x40\xd3\x13\x8d\x81\xa3\x83\x8e\xa1\x57\x5b\x27\x6d\xb1\x7b\xcb\x23\x99\x80\x1f\x61\x56\x6c\xb5\xb0\xd7\xb6\xc8\x14\x7b\xe2\x90\x29\xae\xc6\x45\x3a\xc2\x5d\xf6\xbf\xf1\x8e\x3e\xd9\x45\x46\x6e\xfd\x5a\x67\x47\x6b\x6d\xd1\xe8\x47\xda\xaf\x5d\xbb\xd5\x64\xcd\x1e\x39\xd3\x3c\x06\x3e\xfc\xff\x70\x17\x25\x1e\xbc\xb5\xdd\x9d\xcb\xbf\xb5\x2d\x11\xe6\x45\xab\xa7\xda\xae\xb4\x2d\xde\x2f\xf3\xb7\x76\xb1\x5e\xfe\x22\x25\x8d\x88\x27\xd9\x4d\xeb\x9e\x0d\x75\x86\x37\x58\xa3\x57\xf6\xb6\xbf\xc8\x3f\x10\x4d\x7e\x07\x00\x00\xff\xff\x4c\xf6\xd7\x57\xd8\x07\x00\x00"),
		},
		"/orchestrations/kappnav": &vfsgen۰DirInfo{
			name:    "kappnav",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 490305874, time.UTC),
		},
		"/orchestrations/kappnav/.DS_Store": &vfsgen۰CompressedFileInfo{
			name:             ".DS_Store",
			modTime:          time.Date(2019, 9, 12, 18, 21, 18, 737072990, time.UTC),
			uncompressedSize: 6148,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x98\xb1\x6a\xc3\x30\x10\x86\xef\x14\x0d\x82\x42\xd1\xd8\x51\x4f\x50\xea\x37\x50\x83\xfa\x04\xd9\x3a\x15\x37\x90\x0c\x31\xf6\xd2\xdd\x5b\x9f\xa0\x0f\xd4\x37\x2b\x42\x3f\x54\x60\xbb\x43\xa1\xd4\x31\xff\x07\xe1\x1b\x4e\x77\x39\x3c\xc8\x77\x16\x11\xdd\xbf\x1d\x1b\x11\x2f\x22\x4e\x8a\xe5\x5d\x66\x71\xf8\x4d\x30\xb0\xcd\xf5\x72\x8d\xe6\xf0\xda\x77\xc3\x7c\x95\x05\x72\xee\x4e\x1e\xe4\x5e\x9a\xcb\x69\x92\x8f\x48\xd7\xa7\xd4\x5e\xfa\x36\x37\xf2\x7c\x3a\xdf\xb6\x1f\x9f\x8f\x75\xf4\xf8\x43\x74\x38\xff\xa2\x2b\x42\x08\x21\x64\xbb\x68\x91\xbb\xf9\xef\x46\x08\x21\xab\x23\xdf\x0f\x01\x8e\xf0\x58\xac\x88\x1b\xd8\x56\x39\x1e\x0e\x70\x84\xc7\x62\xc5\x39\x03\x5b\xd8\xc1\x1e\x0e\x70\x84\xc7\x62\x5c\x5a\x8a\xe5\x43\xf1\xcf\x8a\x0d\x45\x3d\x1c\xe0\xf8\x37\xcf\x86\x90\x6b\x67\x57\xe4\xf3\xfb\xff\x69\x79\xff\x27\x84\x6c\x18\xb5\xe9\x90\xf6\xdf\x0b\xc1\x04\x83\x41\xe0\xa5\x4e\x5a\x18\x02\x4c\xf9\x58\x78\x57\x9d\xe3\x20\x40\xc8\xca\xf8\x0a\x00\x00\xff\xff\xef\x21\x97\x94\x04\x18\x00\x00"),
		},
		"/orchestrations/kappnav/0.1": &vfsgen۰DirInfo{
			name:    "0.1",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 490393743, time.UTC),
		},
		"/orchestrations/kappnav/0.1/kappnav-0.1.0.yaml": &vfsgen۰CompressedFileInfo{
			name:             "kappnav-0.1.0.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 490992673, time.UTC),
			uncompressedSize: 1712,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x53\xc1\x6e\xdb\x38\x10\xbd\xeb\x2b\x06\xd9\x43\x80\x05\xa4\xc4\xb7\x05\x6f\xda\x24\xbb\x05\x92\x3a\x86\x92\xb6\xc7\x62\x44\x4f\x2c\x36\x14\x49\x90\x43\xa5\x69\xd1\x7f\x2f\x24\x4b\x96\x1c\xcb\x49\x2e\xd5\x4d\xc3\xe1\x7b\x33\xef\x3d\xa2\x53\x9f\xc9\x07\x65\x8d\x00\x74\x8a\xbe\x33\x99\xf6\x2f\x64\x8f\xff\x84\x4c\xd9\xb3\x66\x51\x12\xe3\x22\x79\x54\x66\x2d\xe0\x22\x06\xb6\x75\x41\xc1\x46\x2f\xe9\x92\x1e\x94\x51\xac\xac\x49\x6a\x62\x5c\x23\xa3\x48\x00\x0c\xd6\x24\xe0\x11\x9d\x33\xd8\x84\x4c\x56\xe8\x39\x64\x15\xe9\xba\x07\x4d\x82\x23\xd9\x76\x6e\xbc\x8d\x4e\xc0\x4c\xc7\x16\x25\xb4\x4d\x00\x5b\xee\xeb\xdc\xb9\x25\x36\x5d\x45\xab\xc0\xd7\xd3\xea\x8d\x0a\xdc\x9d\x38\x1d\x3d\xea\x91\xbe\x2b\x06\x65\x36\x51\xa3\xdf\x95\x13\x80\x20\xad\x23\x01\xcb\x96\xc6\xa1\xa4\x75\x5b\x8b\xa5\xef\x77\xeb\xa9\x03\x23\xc7\x20\xe0\xe7\xaf\x04\xa0\x19\x94\x6a\x16\xa8\x5d\x85\x8b\xb1\xd6\xb5\xa7\xfd\xea\x93\x63\x80\x40\xbe\xa1\xb5\x00\xf6\x91\x7a\x48\xeb\x71\x43\x7d\x25\x4d\xd3\x64\x6a\x42\x33\x48\x7d\x47\xbe\x51\x92\x72\x29\x6d\x34\x3c\x23\x70\x2b\x57\x6a\x1d\x79\x64\xeb\x0f\x70\x7c\x89\x32\xc3\xc8\x95\xf5\xea\x07\xb6\x1e\x8d\x8e\x0e\x66\xea\x18\x98\x7c\x61\x35\xbd\x09\xef\xa3\x6e\x25\x49\xdb\x90\xfc\xdf\xba\xd6\x2f\x7c\xfa\xf7\x69\x02\xb0\x27\xda\x50\x6c\xc8\x97\x93\x42\x0a\xc6\x9a\x21\x39\x9f\x8a\x9b\x57\x7b\xd3\xf4\x70\xc6\x7f\x95\x59\x2b\xb3\x79\xff\x96\x6f\xed\x14\x62\xf9\x8d\x24\x77\x6b\xcd\x6a\x3e\x7f\xad\x8f\x66\x9b\x99\x31\x4f\xde\x6a\x2a\xe8\xa1\xa5\x3a\x14\xf7\x18\xce\xa0\xe5\x2b\x7b\x1c\xf8\x8a\xce\x85\xd1\xc2\x4b\x72\xda\x3e\xd7\xf4\x8e\x80\x0c\x6f\xce\x93\xd3\x4a\x62\x10\xd0\xe6\x33\x90\x26\xc9\xd6\x6f\xd3\x5e\x23\xcb\xea\x06\x4b\xd2\x7d\xfc\x8f\x4d\xce\x54\x3b\x8d\x4c\xfd\xb5\x09\x75\xf7\x3a\xf7\x10\x8e\x61\x00\x0c\x23\x0d\xcf\x64\x94\x7e\x79\xe4\x06\x80\xb4\x86\x51\x19\xf2\x13\xfc\xf4\x28\xc3\xf6\xfb\x0b\x0a\x72\x1a\x25\x01\x57\x2a\xc0\x93\xe2\x0a\xb8\x22\x28\xa3\xd2\x0c\xaa\xc6\x0d\x75\x08\x93\x2b\x5d\x71\x67\xef\xd9\x00\x2a\xce\xb3\x45\x76\xfe\xb2\x6f\x15\xb5\x5e\x59\xad\xe4\xb3\x80\x5c\x3f\xe1\x73\x98\x74\x90\x69\xc4\xe4\x77\x9c\xf6\x4b\x7e\x7f\xf1\xe1\xeb\x32\xff\x78\x75\xb7\xca\x2f\xae\xf6\x7a\x00\x1a\xd4\x91\xfe\xf3\xb6\x16\x2f\x0e\x00\x1e\x14\xe9\x75\x1f\xb6\xd9\xb3\x15\x72\x25\x76\xae\x64\xbb\xb8\xce\x8e\xb1\xba\xbd\xec\x86\xf8\xb3\xfc\xb3\xd4\xb7\xab\xab\x22\xbf\xbf\x2d\x8e\xf2\x0b\x38\xd9\xf3\xf4\x24\xf9\x1d\x00\x00\xff\xff\xb6\xcf\x8f\x3d\xb0\x06\x00\x00"),
		},
		"/orchestrations/landing": &vfsgen۰DirInfo{
			name:    "landing",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 491145052, time.UTC),
		},
		"/orchestrations/landing/.DS_Store": &vfsgen۰CompressedFileInfo{
			name:             ".DS_Store",
			modTime:          time.Date(2019, 9, 10, 11, 39, 56, 368314278, time.UTC),
			uncompressedSize: 6148,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\xd8\x31\x0a\x02\x31\x10\x85\xe1\x37\x31\x45\xc0\x26\xa5\x65\x1a\x0f\xe0\x0d\xc2\xb2\x9e\xc0\x0b\x58\x78\x05\xfb\x1c\x5d\x96\x79\x60\x60\xd5\x4e\x8c\xcb\xfb\x40\xfe\x05\x37\x2a\x16\x31\x23\x00\x9b\xee\xb7\x13\x90\x01\x24\x78\x71\xc4\x4b\x89\x8f\x95\xd0\x5d\x1b\x5f\x43\x44\x44\x44\xc6\x66\x9e\xb4\xff\xf5\x07\x11\x91\xe1\x2c\xfb\x43\x61\x2b\xdb\xbc\xc6\xe7\x03\x1b\xbb\x35\x99\x2d\x6c\x65\x9b\xd7\x78\x5f\x60\x23\x9b\xd8\xcc\x16\xb6\xb2\xcd\xcb\x4d\xcb\x38\x7c\x18\xdf\xd9\x38\xa1\x18\xa7\x10\x2b\x6c\xfd\xce\x77\x23\xf2\xef\x76\x9e\xbc\xfc\xfe\x9f\xdf\xcf\xff\x22\xb2\x61\x16\xe7\xcb\x3c\x3d\x07\x82\xf5\x0d\x00\xae\xdd\xf5\xa7\x43\x40\xf0\x3f\x0b\x0f\xdd\x5a\x1d\x04\x44\x06\xf3\x08\x00\x00\xff\xff\x6a\x00\x88\x6d\x04\x18\x00\x00"),
		},
		"/orchestrations/landing/0.1": &vfsgen۰DirInfo{
			name:    "0.1",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 491237586, time.UTC),
		},
		"/orchestrations/landing/0.1/kabanero-landing.yaml": &vfsgen۰CompressedFileInfo{
			name:             "kabanero-landing.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 491313479, time.UTC),
			uncompressedSize: 423,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x90\x4b\x4e\xc4\x30\x0c\x86\xf7\x39\x85\x2f\x90\x41\x88\xd9\x90\x1d\xe2\x02\x23\x40\xec\x4d\x6a\xda\x68\x5a\xdb\xb2\xdd\x39\x3f\x4a\x2b\x21\x9e\x12\xec\x92\xff\x21\x7d\xbf\x51\xdb\x33\x99\x37\xe1\x02\x97\xeb\x74\x6e\x3c\x14\x78\x24\xbb\xb4\x4a\x69\xa1\xc0\x01\x03\x4b\x02\x60\x5c\xa8\xc0\x19\x5f\x90\xc9\x24\xcf\xc8\x43\xe3\x31\xb9\x52\xed\xb6\xd3\x4c\x35\xc4\xfa\x1b\x00\x55\x7f\xc8\x02\xa8\x58\x78\x8f\x64\x50\x93\x90\x2a\x73\x81\xa7\xfb\xd3\x56\xea\x66\x81\xe3\xf1\x66\xfb\x05\xda\x48\x71\xda\xb4\xdb\x2e\xe6\x9c\xd3\x47\x5a\x93\x35\xe8\x20\x4a\xec\x53\x7b\x8d\x43\x93\xab\xf7\x01\x0f\xdd\xfb\x0f\x7e\xc8\x0e\xfe\x79\x7f\x57\x7e\x29\x02\xc4\xec\x7b\x27\xc8\x96\xc6\x18\x1b\x95\xa2\x7b\x4c\x26\xeb\x38\x7d\x23\xfe\x7a\xdf\xbb\x5a\x65\xe5\xf8\x0b\xe7\x5b\x00\x00\x00\xff\xff\x95\x3c\xf2\x88\xa7\x01\x00\x00"),
		},
		"/samples": &vfsgen۰DirInfo{
			name:    "samples",
			modTime: time.Date(2019, 9, 13, 20, 2, 19, 492999864, time.UTC),
		},
		"/samples/collection.yaml": &vfsgen۰FileInfo{
			name:    "collection.yaml",
			modTime: time.Date(2019, 9, 3, 20, 53, 27, 54551690, time.UTC),
			content: []byte("\x61\x70\x69\x56\x65\x72\x73\x69\x6f\x6e\x3a\x20\x6b\x61\x62\x61\x6e\x65\x72\x6f\x2e\x69\x6f\x2f\x76\x31\x61\x6c\x70\x68\x61\x31\x0a\x6b\x69\x6e\x64\x3a\x20\x43\x6f\x6c\x6c\x65\x63\x74\x69\x6f\x6e\x0a\x6d\x65\x74\x61\x64\x61\x74\x61\x3a\x0a\x20\x20\x6e\x61\x6d\x65\x3a\x20\x6a\x61\x76\x61\x2d\x6d\x69\x63\x72\x6f\x70\x72\x6f\x66\x69\x6c\x65\x0a\x73\x70\x65\x63\x3a\x0a\x20\x20\x76\x65\x72\x73\x69\x6f\x6e\x3a\x20\x30\x2e\x30\x2e\x32"),
		},
		"/samples/default.yaml": &vfsgen۰CompressedFileInfo{
			name:             "default.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 491951406, time.UTC),
			uncompressedSize: 235,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4c\xcd\x41\x6e\x03\x31\x08\x85\xe1\xbd\x4f\xc1\x05\x62\x3a\x5d\xfa\x0a\xdd\x77\x4f\x6c\xd4\x41\xc1\x60\xd9\x24\x6d\x6f\x5f\xa9\xca\x28\xb3\xfd\xd1\xc7\xa3\x21\x9f\x3c\x97\xb8\x15\xb8\xd1\x95\x8c\xa7\x67\x71\x7c\x6c\xa4\x63\xa7\x2d\xdd\xc4\x5a\x81\x8f\xe7\x29\x75\x0e\x6a\x14\x54\x12\x80\x51\xe7\x97\x4a\x6b\x70\x2d\x29\x01\x54\x57\xe5\x1a\xe2\xb6\x0a\x24\x00\x80\xc9\xc3\x97\x84\x4f\xe1\x23\x5d\x9e\xbc\xb2\xc5\x24\xfd\x6f\x00\xf7\xa9\x05\xf6\x88\xb1\x0a\xe2\x97\xc4\x7e\xbf\xe6\xea\x1d\x8f\x91\x8b\x38\x9e\xde\xe3\x64\x65\x5a\xbc\xb0\xf9\xb7\xa9\x53\xc3\xc7\x5b\xde\xf2\xfb\x09\x58\xe3\x9f\xfc\x4b\x5d\xff\x02\x00\x00\xff\xff\xea\xc4\xe5\xca\xeb\x00\x00\x00"),
		},
		"/samples/full.yaml": &vfsgen۰CompressedFileInfo{
			name:             "full.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 492845113, time.UTC),
			uncompressedSize: 1398,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x94\x3f\x73\xdb\x3c\x0c\x87\x77\x7d\x0a\x5c\xbc\xbe\xb6\xe2\x77\xd4\x96\x36\x5b\xef\xda\xa1\xb9\xee\x90\x08\x4b\x38\x93\x84\x8e\x80\xe4\xf8\xdb\xf7\x48\xc9\x7f\x92\x36\x5d\x3a\x74\xb4\x48\x3c\xf8\xe1\x39\xd0\x1b\x78\x19\x58\x81\x5e\x31\x8c\x9e\xc0\x51\x90\xa8\x96\xd0\x48\x01\xbd\x07\x39\x80\x0d\x04\xa3\xa8\x72\xeb\x09\x3a\x89\x07\xee\xa7\x84\xc6\x12\x81\x3c\x05\x8a\xa6\xd5\xa6\xda\xc0\x57\x31\x02\x1b\xd0\x40\x25\xd0\xa5\x12\xcd\x12\xb7\x53\xe6\x05\x3c\x43\x14\x83\x80\x47\x02\xa5\xa8\x04\x1c\xa1\x93\xd0\x72\x5c\x78\x27\xb6\x01\xc4\x06\x4a\x77\x75\xd5\x06\x30\xba\x85\x6c\x39\xac\x2e\x59\x03\xf7\x83\x15\x60\x4b\x90\xa6\x88\x39\x1f\xea\x96\xb5\xc2\x91\x7f\x50\x52\x96\xd8\xc0\x11\x5b\x8c\x94\x64\xc7\x52\xcf\x7b\xf4\xe3\x80\xfb\xea\xc8\xd1\x35\xf0\x65\x3d\xaa\x02\x19\x3a\x34\x6c\x2a\x80\x88\x81\x6e\x55\x95\x8e\xd4\xe5\xcf\x59\x14\xc1\xe8\xd1\x0e\x92\x02\xcc\x0b\x1e\x1c\x19\xa5\xc0\x91\xb4\x4c\xeb\x48\x39\x91\xbb\x1e\x1f\x24\x15\x8d\x9d\x84\x51\x62\x56\xf5\x1f\xb4\x53\x9e\x43\x94\x0a\xb5\xc3\x98\xf3\xcb\x4c\x29\xb1\xa3\x08\x1c\x1d\xcf\xec\x26\xf4\xfe\x0c\xa8\x70\x22\xef\x2b\xb8\x10\x1b\x78\x78\xdc\xed\x77\xff\x3f\x54\x15\x00\x8e\xa3\x8a\x3b\x6f\x65\xa4\x84\x26\x29\xe7\xcc\xcc\x6f\x2b\x6c\xc9\x74\xcd\x7c\x62\x97\xbd\x9b\x71\xec\xb5\x44\xbb\xc4\x94\xb8\x98\xbd\xc6\x2c\xa0\xb7\x2d\xf7\xa5\xe5\xaf\x78\x0e\xd8\x67\xef\x80\xa0\x34\x62\x5e\x1d\x48\x34\x8a\xb2\x49\x3a\x83\x24\x30\xec\x4b\xe5\xed\x6b\x73\x89\x5e\xbf\x1f\xa1\x5c\x34\xec\xd7\xa6\x8f\x7f\x6c\x3a\x25\x2e\xa7\xe5\xd7\xc7\xcc\xa6\x90\x32\xe8\x88\xe3\x18\x71\x5e\x3c\x51\xd9\x98\x06\x2c\x4d\x94\x0f\x3b\xcf\xdf\x29\xcd\xdc\x91\xfe\x5e\xe4\xaa\xee\x1f\x9b\xbb\x6c\xe6\x96\xa5\xee\x3c\x6f\x75\xcd\xfc\x37\xe6\x3e\x62\xde\xcc\x79\x8c\x8e\x63\x7f\x11\xf3\xf2\xe9\xb9\x2a\xce\xc4\x7b\xea\xf2\xdb\xd5\x06\xd6\xb3\x27\xf0\xac\xb6\xbc\x7f\xd1\xbb\x91\x98\x14\x4e\x03\x77\x03\x60\xca\x36\x31\x75\x03\xb9\xa2\xf3\x8e\xf3\x76\x60\xa6\x0b\x78\xbb\xbe\x4c\x8e\xdd\xd4\x5e\x57\x05\x60\x4a\xbe\x81\xc1\x6c\xd4\xa6\xae\x7b\xb6\x61\x6a\x77\x9d\x84\xfa\xcd\x4c\x37\x7c\x9d\xc8\x13\x2a\x69\xed\xe4\x14\xbd\xa0\xab\xe7\xf2\xa2\xee\x0a\xa2\xa3\xd7\xdd\x19\x83\x5f\x5b\x6c\xe0\xa9\x33\x9e\x97\x3f\x45\x70\x74\xc0\xc9\x5b\x5e\x87\x3c\xe4\xfb\xe8\x00\xb8\x5e\x7e\x5e\x2e\x7e\xbe\x77\x94\x97\x0d\xaa\x9f\x01\x00\x00\xff\xff\xac\x49\x50\xc6\x76\x05\x00\x00"),
		},
		"/samples/override_software_versions.yaml": &vfsgen۰CompressedFileInfo{
			name:             "override_software_versions.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 493458175, time.UTC),
			uncompressedSize: 1243,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xcc\x93\x31\x6f\xc2\x30\x10\x85\x77\xff\x8a\x13\x5d\x9b\x40\xe8\x96\xad\x73\x07\x96\xaa\x9d\x2f\xf8\x44\x4e\x38\x67\xcb\x39\x82\xf2\xef\xab\x38\x09\x30\x94\x42\x25\x2a\x75\x8a\xe4\x3c\xbf\xbb\xef\xe9\x19\x03\x7f\x50\x6c\xd9\x4b\x09\x7b\xac\x50\x28\xfa\x9c\xfd\xb2\x2b\xd0\x85\x1a\x0b\xb3\x67\xb1\x25\xbc\x4d\xbf\x4c\x43\x8a\x16\x15\x4b\x03\x20\xd8\xd0\xf9\x56\x61\xda\x40\xdb\xe1\xbc\x9b\x0d\x57\x79\x91\xaf\x0c\x80\x01\xc0\x10\x5a\x6f\xfb\x4d\xa0\x88\xea\xe3\x20\x03\x78\x82\x4d\x47\x31\xb2\xa5\x16\xb4\x26\x60\x51\x8a\x82\xce\xf5\x60\x49\x29\x36\x2c\x64\xe1\x75\xbc\x3a\xdb\xc2\x91\xb5\x4e\xf2\x10\x7d\xc7\x96\x2c\x74\xe8\x0e\x94\x1c\x4f\xa3\x17\x69\xf6\xc2\x64\x59\x66\xfe\x82\xf1\x01\x3c\xdc\xe0\x8e\x7e\xa4\x49\x8a\x72\x9e\xb5\x9c\xbe\x99\x9f\x87\x8e\xf9\x3e\x16\x71\x7d\x3f\xa2\x17\xd7\xa7\xd5\x23\x05\xdf\xb2\xfa\xd8\xc3\xa1\x25\x0b\xd5\x78\x3c\x19\xc0\xbc\x6f\x3e\x39\xbc\xd7\x04\x8a\x3b\x38\xb2\x73\x50\x5d\x89\x29\x69\xcf\xc6\xd7\x53\xf8\x07\xfc\x03\x4c\x02\x67\xf9\x16\x1c\x2c\x05\xe7\xfb\x86\x44\x2f\x22\xb8\x48\xed\x76\x12\x8a\xbb\xa9\xd4\xeb\x47\x97\xfa\xe5\x16\xf1\x67\x4d\x02\x95\xd7\x7a\xaa\x2c\x8a\xbd\x5c\xde\x9f\x02\xc1\x38\xd4\x98\x5a\x12\x7d\x1e\x1f\xc0\xa8\x57\x8d\x5c\x1d\x94\x26\xbf\x44\xab\xb8\x4f\xe2\x2d\x59\x92\xed\x6f\xfa\x7e\x6f\x31\xbe\x02\x00\x00\xff\xff\x69\x23\x65\x0a\xdb\x04\x00\x00"),
		},
		"/samples/simple.yaml": &vfsgen۰FileInfo{
			name:    "simple.yaml",
			modTime: time.Date(2019, 7, 17, 14, 52, 51, 831521466, time.UTC),
			content: []byte("\x61\x70\x69\x56\x65\x72\x73\x69\x6f\x6e\x3a\x20\x6b\x61\x62\x61\x6e\x65\x72\x6f\x2e\x69\x6f\x2f\x76\x31\x61\x6c\x70\x68\x61\x31\x0a\x6b\x69\x6e\x64\x3a\x20\x4b\x61\x62\x61\x6e\x65\x72\x6f\x0a\x6d\x65\x74\x61\x64\x61\x74\x61\x3a\x0a\x20\x20\x6e\x61\x6d\x65\x3a\x20\x6b\x61\x62\x61\x6e\x65\x72\x6f\x0a\x73\x70\x65\x63\x3a\x20\x7b\x7d\x0a"),
		},
		"/versions.yaml": &vfsgen۰CompressedFileInfo{
			name:             "versions.yaml",
			modTime:          time.Date(2019, 9, 13, 20, 2, 19, 494200731, time.UTC),
			uncompressedSize: 1596,

			compressedContent: []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x92\xcd\x8e\xdb\x3a\x0c\x85\xf7\x7a\x0a\xc2\xd9\xc6\xf9\x99\xbb\xf3\x2b\x5c\xb4\xdd\x0c\xd0\xb5\x22\xd1\x31\x51\x59\x54\x45\x39\x41\xde\xbe\x90\x6d\x39\x71\x9a\xc5\x60\x0a\xb4\x3b\x81\x14\x0f\xbf\x43\x72\x03\x5f\x74\x10\xf8\x5f\x9f\xb4\xc7\xc8\x10\x9c\x4e\x2d\xc7\x1e\x2e\x18\x85\xd8\x43\x62\x88\xe8\x74\x42\x0b\x86\xfb\xc0\x1e\x7d\x2a\x49\x51\x1b\x18\xbc\xc5\x28\x89\xd9\xc2\xe9\x06\xa9\x43\xe0\xd4\x61\x04\x0e\x18\x75\xe2\x28\x3b\x50\x6a\x03\x5f\x39\xe1\x98\x6d\xd9\x39\xbe\x92\x3f\x83\xe0\xcf\x01\xbd\xc1\x66\x8c\x2f\x04\xa5\x10\x4e\x68\xb8\x47\x01\x7d\xd5\x11\xd5\x06\xb8\x05\x0d\x1e\xaf\x0b\x1a\xb7\xeb\xca\xc2\xbe\x05\xf2\x10\x74\x4c\x99\x08\x2d\xa5\xdc\x2d\x75\x24\xd0\x92\xcb\x4a\xdf\xbc\x99\x60\x16\x48\xd0\x11\x21\xb2\x73\x68\x81\x87\xb4\x1d\xb3\xda\xf6\xe4\x49\xd2\x84\x63\xb4\x07\xd3\x31\x0b\xe6\x99\x0c\x92\x85\xf2\xaf\x4c\xb4\x20\x14\xb4\x55\xe3\x07\xc4\x88\xc2\x43\x34\xb8\x83\xf7\xcc\x63\xf4\x20\x98\xa7\x38\xcd\xac\x64\x25\x77\x38\x21\x0c\xc1\x8e\x83\x1f\x64\x71\xd0\xeb\x10\xc8\x9f\xf3\x48\xbf\x77\xe8\xd7\xea\xe4\x25\xe9\xec\xcd\x32\x0a\x78\x4e\x20\x01\x0d\xb5\x37\xb8\x76\x64\xba\xc7\xc1\x2d\x35\x93\x97\xed\x68\x86\x04\x48\x46\xc9\x8b\x76\x03\x2a\x8b\xad\x1e\x5c\x6a\xa0\x3a\xec\x8e\xbb\xb7\x2a\x77\x7d\xe7\x00\x0e\x2f\xe8\x9a\xf9\x2e\xe4\x37\xf7\x23\xbe\x70\x9b\xf2\xe2\xee\xb7\xf2\x63\xfe\xd6\xa8\xba\x04\x67\xe5\x63\xa5\xa0\x5c\x59\x5d\xfe\x37\xa0\x00\x00\x74\x08\xc2\xf6\x56\x97\x55\xcd\x25\x87\x6a\xcc\x1a\x47\xb5\x60\xbc\x90\x41\x59\x67\x9c\xf6\x96\xfc\xf9\x1e\x7c\x6e\xfa\xf6\x2f\x9a\xfe\xf7\x37\x9a\xaa\xd2\xa0\xec\xa0\x51\x2f\xd4\x15\xc0\x33\xdd\x2c\xc8\xd1\x74\x38\x9e\xfd\x44\x57\xad\x03\xfb\x67\xa9\xfd\x61\xda\x20\x00\x59\xf4\x89\x5a\xc2\x28\xcd\x18\xc8\x66\x03\x0b\x25\x8e\xb7\x06\xaa\xb9\xb2\x28\x54\xf3\x9f\xa4\x1f\xe9\xef\x8e\x3e\x8f\x38\x2b\x4c\x64\xea\x79\x7a\x2f\x74\x8f\x1f\xd3\x7d\xd4\xf9\xb8\xed\x72\xf9\x35\xf1\x7e\x79\x1b\x47\x2f\xfd\x7f\xda\xf3\x9f\xb1\x3d\x80\x71\xdf\x6b\x6f\x6b\x47\x1e\x17\xc1\x57\xa8\xbf\x02\x00\x00\xff\xff\x19\x14\xce\x08\x3c\x06\x00\x00"),
		},
	}
	fs["/"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations"].(os.FileInfo),
		fs["/samples"].(os.FileInfo),
		fs["/versions.yaml"].(os.FileInfo),
	}
	fs["/orchestrations"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/.DS_Store"].(os.FileInfo),
		fs["/orchestrations/appsody-operator"].(os.FileInfo),
		fs["/orchestrations/cli-services"].(os.FileInfo),
		fs["/orchestrations/kappnav"].(os.FileInfo),
		fs["/orchestrations/landing"].(os.FileInfo),
	}
	fs["/orchestrations/appsody-operator"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/appsody-operator/.DS_Store"].(os.FileInfo),
		fs["/orchestrations/appsody-operator/0.1"].(os.FileInfo),
	}
	fs["/orchestrations/appsody-operator/0.1"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/appsody-operator/0.1/appsody.yaml"].(os.FileInfo),
	}
	fs["/orchestrations/cli-services"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/cli-services/.DS_Store"].(os.FileInfo),
		fs["/orchestrations/cli-services/0.1"].(os.FileInfo),
	}
	fs["/orchestrations/cli-services/0.1"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/cli-services/0.1/kabanero-cli.yaml"].(os.FileInfo),
	}
	fs["/orchestrations/kappnav"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/kappnav/.DS_Store"].(os.FileInfo),
		fs["/orchestrations/kappnav/0.1"].(os.FileInfo),
	}
	fs["/orchestrations/kappnav/0.1"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/kappnav/0.1/kappnav-0.1.0.yaml"].(os.FileInfo),
	}
	fs["/orchestrations/landing"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/landing/.DS_Store"].(os.FileInfo),
		fs["/orchestrations/landing/0.1"].(os.FileInfo),
	}
	fs["/orchestrations/landing/0.1"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/orchestrations/landing/0.1/kabanero-landing.yaml"].(os.FileInfo),
	}
	fs["/samples"].(*vfsgen۰DirInfo).entries = []os.FileInfo{
		fs["/samples/collection.yaml"].(os.FileInfo),
		fs["/samples/default.yaml"].(os.FileInfo),
		fs["/samples/full.yaml"].(os.FileInfo),
		fs["/samples/override_software_versions.yaml"].(os.FileInfo),
		fs["/samples/simple.yaml"].(os.FileInfo),
	}

	return fs
}()

type vfsgen۰FS map[string]interface{}

func (fs vfsgen۰FS) Open(path string) (http.File, error) {
	path = pathpkg.Clean("/" + path)
	f, ok := fs[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}

	switch f := f.(type) {
	case *vfsgen۰CompressedFileInfo:
		gr, err := gzip.NewReader(bytes.NewReader(f.compressedContent))
		if err != nil {
			// This should never happen because we generate the gzip bytes such that they are always valid.
			panic("unexpected error reading own gzip compressed bytes: " + err.Error())
		}
		return &vfsgen۰CompressedFile{
			vfsgen۰CompressedFileInfo: f,
			gr:                        gr,
		}, nil
	case *vfsgen۰FileInfo:
		return &vfsgen۰File{
			vfsgen۰FileInfo: f,
			Reader:          bytes.NewReader(f.content),
		}, nil
	case *vfsgen۰DirInfo:
		return &vfsgen۰Dir{
			vfsgen۰DirInfo: f,
		}, nil
	default:
		// This should never happen because we generate only the above types.
		panic(fmt.Sprintf("unexpected type %T", f))
	}
}

// vfsgen۰CompressedFileInfo is a static definition of a gzip compressed file.
type vfsgen۰CompressedFileInfo struct {
	name              string
	modTime           time.Time
	compressedContent []byte
	uncompressedSize  int64
}

func (f *vfsgen۰CompressedFileInfo) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("cannot Readdir from file %s", f.name)
}
func (f *vfsgen۰CompressedFileInfo) Stat() (os.FileInfo, error) { return f, nil }

func (f *vfsgen۰CompressedFileInfo) GzipBytes() []byte {
	return f.compressedContent
}

func (f *vfsgen۰CompressedFileInfo) Name() string       { return f.name }
func (f *vfsgen۰CompressedFileInfo) Size() int64        { return f.uncompressedSize }
func (f *vfsgen۰CompressedFileInfo) Mode() os.FileMode  { return 0444 }
func (f *vfsgen۰CompressedFileInfo) ModTime() time.Time { return f.modTime }
func (f *vfsgen۰CompressedFileInfo) IsDir() bool        { return false }
func (f *vfsgen۰CompressedFileInfo) Sys() interface{}   { return nil }

// vfsgen۰CompressedFile is an opened compressedFile instance.
type vfsgen۰CompressedFile struct {
	*vfsgen۰CompressedFileInfo
	gr      *gzip.Reader
	grPos   int64 // Actual gr uncompressed position.
	seekPos int64 // Seek uncompressed position.
}

func (f *vfsgen۰CompressedFile) Read(p []byte) (n int, err error) {
	if f.grPos > f.seekPos {
		// Rewind to beginning.
		err = f.gr.Reset(bytes.NewReader(f.compressedContent))
		if err != nil {
			return 0, err
		}
		f.grPos = 0
	}
	if f.grPos < f.seekPos {
		// Fast-forward.
		_, err = io.CopyN(ioutil.Discard, f.gr, f.seekPos-f.grPos)
		if err != nil {
			return 0, err
		}
		f.grPos = f.seekPos
	}
	n, err = f.gr.Read(p)
	f.grPos += int64(n)
	f.seekPos = f.grPos
	return n, err
}
func (f *vfsgen۰CompressedFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.seekPos = 0 + offset
	case io.SeekCurrent:
		f.seekPos += offset
	case io.SeekEnd:
		f.seekPos = f.uncompressedSize + offset
	default:
		panic(fmt.Errorf("invalid whence value: %v", whence))
	}
	return f.seekPos, nil
}
func (f *vfsgen۰CompressedFile) Close() error {
	return f.gr.Close()
}

// vfsgen۰FileInfo is a static definition of an uncompressed file (because it's not worth gzip compressing).
type vfsgen۰FileInfo struct {
	name    string
	modTime time.Time
	content []byte
}

func (f *vfsgen۰FileInfo) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("cannot Readdir from file %s", f.name)
}
func (f *vfsgen۰FileInfo) Stat() (os.FileInfo, error) { return f, nil }

func (f *vfsgen۰FileInfo) NotWorthGzipCompressing() {}

func (f *vfsgen۰FileInfo) Name() string       { return f.name }
func (f *vfsgen۰FileInfo) Size() int64        { return int64(len(f.content)) }
func (f *vfsgen۰FileInfo) Mode() os.FileMode  { return 0444 }
func (f *vfsgen۰FileInfo) ModTime() time.Time { return f.modTime }
func (f *vfsgen۰FileInfo) IsDir() bool        { return false }
func (f *vfsgen۰FileInfo) Sys() interface{}   { return nil }

// vfsgen۰File is an opened file instance.
type vfsgen۰File struct {
	*vfsgen۰FileInfo
	*bytes.Reader
}

func (f *vfsgen۰File) Close() error {
	return nil
}

// vfsgen۰DirInfo is a static definition of a directory.
type vfsgen۰DirInfo struct {
	name    string
	modTime time.Time
	entries []os.FileInfo
}

func (d *vfsgen۰DirInfo) Read([]byte) (int, error) {
	return 0, fmt.Errorf("cannot Read from directory %s", d.name)
}
func (d *vfsgen۰DirInfo) Close() error               { return nil }
func (d *vfsgen۰DirInfo) Stat() (os.FileInfo, error) { return d, nil }

func (d *vfsgen۰DirInfo) Name() string       { return d.name }
func (d *vfsgen۰DirInfo) Size() int64        { return 0 }
func (d *vfsgen۰DirInfo) Mode() os.FileMode  { return 0755 | os.ModeDir }
func (d *vfsgen۰DirInfo) ModTime() time.Time { return d.modTime }
func (d *vfsgen۰DirInfo) IsDir() bool        { return true }
func (d *vfsgen۰DirInfo) Sys() interface{}   { return nil }

// vfsgen۰Dir is an opened dir instance.
type vfsgen۰Dir struct {
	*vfsgen۰DirInfo
	pos int // Position within entries for Seek and Readdir.
}

func (d *vfsgen۰Dir) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekStart {
		d.pos = 0
		return 0, nil
	}
	return 0, fmt.Errorf("unsupported Seek in directory %s", d.name)
}

func (d *vfsgen۰Dir) Readdir(count int) ([]os.FileInfo, error) {
	if d.pos >= len(d.entries) && count > 0 {
		return nil, io.EOF
	}
	if count <= 0 || count > len(d.entries)-d.pos {
		count = len(d.entries) - d.pos
	}
	e := d.entries[d.pos : d.pos+count]
	d.pos += count
	return e, nil
}
