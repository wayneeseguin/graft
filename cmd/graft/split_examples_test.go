package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSplitExamples(t *testing.T) {
	Convey("Split operator examples should merge correctly", t, func() {
		// Get the project root directory
		wd, err := os.Getwd()
		So(err, ShouldBeNil)

		// Go up to project root if we're in cmd/graft
		if filepath.Base(wd) == "graft" && filepath.Base(filepath.Dir(wd)) == "cmd" {
			wd = filepath.Dir(filepath.Dir(wd))
		}

		Convey("Network parsing example", func() {
			filePath := filepath.Join(wd, "examples/split/network-parsing.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check IP octets
			ipOctets := m["network_data"].(map[interface{}]interface{})["ip_octets"].([]interface{})
			So(len(ipOctets), ShouldEqual, 4)
			So(ipOctets[0], ShouldEqual, "192")
			So(ipOctets[1], ShouldEqual, "168")
			So(ipOctets[2], ShouldEqual, "1")
			So(ipOctets[3], ShouldEqual, "100")

			// Check MAC parts
			macParts := m["network_data"].(map[interface{}]interface{})["mac_parts"].([]interface{})
			So(len(macParts), ShouldEqual, 6)
		})

		Convey("Data extraction example", func() {
			filePath := filepath.Join(wd, "examples/split/data-extraction.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check URL params
			params := m["data_extraction"].(map[interface{}]interface{})["params"].([]interface{})
			So(len(params), ShouldEqual, 4)
			So(params[0], ShouldEqual, "host=localhost")
			So(params[1], ShouldEqual, "port=8080")

			// Check JSON fields
			jsonFields := m["data_extraction"].(map[interface{}]interface{})["json_fields"].([]interface{})
			So(len(jsonFields), ShouldEqual, 5)
			So(jsonFields[0], ShouldEqual, "")
			So(jsonFields[1], ShouldEqual, "name:john")
		})

		Convey("Version parsing example", func() {
			filePath := filepath.Join(wd, "examples/split/version-parsing.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check base version
			baseVersion := m["version_parsing"].(map[interface{}]interface{})["base_version"].([]interface{})
			So(len(baseVersion), ShouldEqual, 3)
			So(baseVersion[0], ShouldEqual, "2.1.3")
			So(baseVersion[1], ShouldEqual, "beta.1")
			So(baseVersion[2], ShouldEqual, "build.456")
		})

		Convey("Comprehensive tests example", func() {
			filePath := filepath.Join(wd, "examples/split/comprehensive-tests.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check basic tests
			basicTests := m["basic_tests"].(map[interface{}]interface{})
			commaSplit := basicTests["comma_split"].([]interface{})
			So(len(commaSplit), ShouldEqual, 3)
			So(commaSplit[0], ShouldEqual, "apple")
			So(commaSplit[1], ShouldEqual, "banana")
			So(commaSplit[2], ShouldEqual, "cherry")

			// Check edge cases
			edgeCases := m["edge_cases"].(map[interface{}]interface{})
			unicodeSplit := edgeCases["unicode_split"].([]interface{})
			So(len(unicodeSplit), ShouldEqual, 3)
			So(unicodeSplit[0], ShouldEqual, "Hello")
			So(unicodeSplit[1], ShouldEqual, "World")
			So(unicodeSplit[2], ShouldEqual, "Test")
		})

		Convey("Operator integration example", func() {
			filePath := filepath.Join(wd, "examples/split/operator-integration.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check grab integration
			grabInt := m["grab_integration"].(map[interface{}]interface{})
			connParams := grabInt["connection_params"].([]interface{})
			So(len(connParams), ShouldEqual, 4)
			So(connParams[0], ShouldEqual, "host=localhost")
			So(connParams[1], ShouldEqual, "port=5432")

			// Check concat integration
			concatInt := m["concat_integration"].(map[interface{}]interface{})
			splitFruits := concatInt["split_fruits"].([]interface{})
			So(len(splitFruits), ShouldEqual, 3)
			So(splitFruits[0], ShouldEqual, "apple")
		})

		Convey("Regex patterns example", func() {
			filePath := filepath.Join(wd, "examples/split/regex-patterns.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check basic regex
			basicRegex := m["basic_regex"].(map[interface{}]interface{})
			ipOctets := basicRegex["ip_octets"].([]interface{})
			So(len(ipOctets), ShouldEqual, 4)
			So(ipOctets[0], ShouldEqual, "192")

			// Check PCRE features
			pcreFeatures := m["pcre_features"].(map[interface{}]interface{})
			lookbehindSplit := pcreFeatures["lookbehind_split"].([]interface{})
			So(len(lookbehindSplit), ShouldEqual, 3)
			So(lookbehindSplit[0], ShouldEqual, "abc123")
			So(lookbehindSplit[1], ShouldEqual, "def456")
			So(lookbehindSplit[2], ShouldEqual, "ghi")
		})

		Convey("Reversibility tests example", func() {
			filePath := filepath.Join(wd, "examples/split/reversibility-tests.yml")
			files, err := openFiles([]string{filePath})
			So(err, ShouldBeNil)
			defer func() {
				for _, f := range files {
					f.Reader.(*os.File).Close()
				}
			}()

			m, err := mergeAllDocs(files, mergeOpts{})
			So(err, ShouldBeNil)

			// Check perfect reversibility
			perfectRev := m["perfect_reversibility"].(map[interface{}]interface{})

			// Check split and rejoin
			originalCSV := perfectRev["original_csv"].(string)
			rejoinCSV := perfectRev["rejoin_csv"].(string)
			So(rejoinCSV, ShouldEqual, originalCSV)

			// Check empty string cases
			emptyStrings := m["empty_string_cases"].(map[interface{}]interface{})
			splitEmpties := emptyStrings["split_empties"].([]interface{})
			So(len(splitEmpties), ShouldEqual, 6)
			So(splitEmpties[1], ShouldEqual, "")
			So(splitEmpties[3], ShouldEqual, "")
			So(splitEmpties[4], ShouldEqual, "")
		})
	})
}
