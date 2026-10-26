package ffprobe

import "testing"

func TestDetectHDRFormats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		stream probeStream
		want   string
	}{
		{name: "HDR10", stream: probeStream{ColorTransfer: "smpte2084"}, want: "hdr10"},
		{name: "HLG", stream: probeStream{ColorTransfer: "arib-std-b67"}, want: "hlg"},
		{name: "HDR10 plus", stream: probeStream{ColorTransfer: "smpte2084", SideDataList: []probeSideData{{SideDataType: "HDR Dynamic Metadata SMPTE2094-40"}}}, want: "hdr10plus"},
		{name: "Dolby Vision", stream: probeStream{ColorTransfer: "smpte2084", SideDataList: []probeSideData{{SideDataType: "DOVI configuration record"}}}, want: "dolby-vision"},
		{name: "SDR", stream: probeStream{ColorTransfer: "bt709"}, want: ""},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := detectHDR(test.stream); got != test.want {
				t.Fatalf("detectHDR() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBitDepthFromPixelFormat(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		format string
		want   int
	}{
		{format: "yuv420p", want: 8},
		{format: "yuv420p10le", want: 10},
		{format: "gbrp12le", want: 12},
		{format: "", want: 0},
	} {
		if got := bitDepthFromPixelFormat(test.format); got != test.want {
			t.Fatalf("bitDepthFromPixelFormat(%q) = %d, want %d", test.format, got, test.want)
		}
	}
}

func TestNormalizeContainerAndCodec(t *testing.T) {
	t.Parallel()
	if got := normalizeContainer("matroska,webm"); got != "mkv" {
		t.Fatalf("normalizeContainer() = %q", got)
	}
	if got := normalizeContainer("mov,mp4,m4a,3gp,3g2,mj2"); got != "mp4" {
		t.Fatalf("normalizeContainer() = %q", got)
	}
	if got := normalizeCodec("dca"); got != "dts" {
		t.Fatalf("normalizeCodec() = %q", got)
	}
	if got := normalizeLanguage("est"); got != "et" {
		t.Fatalf("normalizeLanguage() = %q", got)
	}
}
