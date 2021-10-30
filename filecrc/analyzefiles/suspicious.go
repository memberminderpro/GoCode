package main

import "dacdb.com/GoCode/filecrc/utils"

var (
	suspiciousCt int = 0 // Number of suspicious files encountered
)

// isSuspicious Check to see if the data for the current record is suspicious
// Uses information from the historical data and compares specific
// values such as crc, file size and file date/time to deteremine if the
// file is suspicious
func isSuspicious(data utils.FileInfo, fileData utils.FileInfo) (string, bool) {
	// Build up a set of changes
	violation := ""
	suspicious := false

	// Do some time checking
	// Note: You CANNOT verify the modified date against the create date since windows retains the
	// modified date when copying a file but sets the create date to the time of the copy
	// so with a copied file the modified date is always less than the created date
	if data.GetAccessed().Before(data.GetCreated()) {
		violation = "File times are inconsistent"
	}

	// CRC comparisons cannot be done when not generating CRC
	if !parmExcludeCRC {
		// Rule2: Only the size changed
		if data.GetSize() != fileData.GetSize() && data.GetCRC() == fileData.GetCRC() && data.TimesEqual(fileData) {
			violation = "Only the size has changed"
		}

		// Rule3: Only the CRC changed
		if data.GetCRC() != fileData.GetCRC() && data.GetSize() == fileData.GetSize() && data.TimesEqual(fileData) {
			violation = "Only crc changed"
		}

		// Rule4: CRC and size changed but not modified date
		if data.GetCRC() != fileData.GetCRC() && data.GetSize() != fileData.GetSize() &&
			data.GetModified() == fileData.GetModified() {
			violation = "CRC and size changed but not modified"
		}
	}

	// Any violations are suspicious
	if len(violation) > 0 {
		suspicious = true
	}

	// Rturn the result
	return violation, suspicious
}
