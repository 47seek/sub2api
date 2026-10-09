package tlsfingerprint

// Codex0162 is the fixed Linux 0.162.0 release ClientHello captured by this
// research. Matching the advertisement does not make uTLS an OpenSSL stack.
var Codex0162 = &Profile{
	Name:                "Codex Linux 0.162.0",
	CipherSuites:        []uint16{4866, 4867, 4865, 49196, 49200, 159, 52393, 52392, 52394, 49195, 49199, 158, 49188, 49192, 107, 49187, 49191, 103, 49162, 49172, 57, 49161, 49171, 51, 157, 156, 61, 60, 53, 47},
	Extensions:          []uint16{65281, 0, 11, 10, 35, 22, 23, 13, 43, 45, 51},
	Curves:              []uint16{4588, 29, 23, 30, 24, 25, 256, 257},
	PointFormats:        []uint16{0},
	SignatureAlgorithms: []uint16{2309, 2310, 2308, 1027, 1283, 1539, 2055, 2056, 2074, 2075, 2076, 2057, 2058, 2059, 2052, 2053, 2054, 1025, 1281, 1537, 771, 769, 770, 1026, 1282, 1538},
	SupportedVersions:   []uint16{772, 771},
	KeyShareGroups:      []uint16{4588, 29},
	PSKModes:            []uint16{1},
}
