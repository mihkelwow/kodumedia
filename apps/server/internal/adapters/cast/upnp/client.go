package upnp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mihkelwow/kodumedia/internal/domain"
	"github.com/mihkelwow/kodumedia/internal/ports"
)

const (
	mediaRendererSearchTarget = "urn:schemas-upnp-org:device:MediaRenderer:1"
	avTransportServicePrefix  = "urn:schemas-upnp-org:service:AVTransport:"
	renderingServicePrefix    = "urn:schemas-upnp-org:service:RenderingControl:"
	ssdpAddress               = "239.255.255.250:1900"
)

type Client struct {
	httpClient      *http.Client
	discoverTimeout time.Duration
	cacheTTL        time.Duration

	mu         sync.RWMutex
	devices    map[string]deviceRecord
	cacheUntil time.Time
}

type deviceRecord struct {
	device               domain.CastDevice
	location             *url.URL
	avTransportURL       *url.URL
	avTransportType      string
	renderingControlURL  *url.URL
	renderingControlType string
}

type descriptionRoot struct {
	URLBase string            `xml:"URLBase"`
	Device  descriptionDevice `xml:"device"`
}

type descriptionDevice struct {
	DeviceType      string               `xml:"deviceType"`
	FriendlyName    string               `xml:"friendlyName"`
	Manufacturer    string               `xml:"manufacturer"`
	ModelName       string               `xml:"modelName"`
	ModelNumber     string               `xml:"modelNumber"`
	UDN             string               `xml:"UDN"`
	ServiceList     []descriptionService `xml:"serviceList>service"`
	EmbeddedDevices []descriptionDevice  `xml:"deviceList>device"`
}

type descriptionService struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
			CheckRedirect: func(request *http.Request, _ []*http.Request) error {
				if !isLocalURL(request.Context(), request.URL) {
					return fmt.Errorf("UPnP description redirect left the local network")
				}
				return nil
			},
		},
		discoverTimeout: 1500 * time.Millisecond,
		cacheTTL:        20 * time.Second,
		devices:         make(map[string]deviceRecord),
	}
}

func (c *Client) Discover(ctx context.Context) ([]domain.CastDevice, error) {
	c.mu.RLock()
	if time.Now().Before(c.cacheUntil) {
		devices := publicDevices(c.devices)
		c.mu.RUnlock()
		return devices, nil
	}
	c.mu.RUnlock()

	locations, discoveryErr := discoverLocations(ctx, c.discoverTimeout)
	if discoveryErr != nil && len(locations) == 0 {
		c.mu.RLock()
		cached := publicDevices(c.devices)
		c.mu.RUnlock()
		if len(cached) > 0 {
			return cached, nil
		}
		return nil, discoveryErr
	}

	records := make(chan deviceRecord, len(locations))
	var wait sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	for _, location := range locations {
		location := location
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()
			record, err := c.describe(ctx, location)
			if err == nil {
				records <- record
			}
		}()
	}
	go func() {
		wait.Wait()
		close(records)
	}()

	updated := make(map[string]deviceRecord)
	for record := range records {
		updated[record.device.ID] = record
	}
	if len(updated) == 0 {
		c.mu.RLock()
		cached := publicDevices(c.devices)
		c.mu.RUnlock()
		if len(cached) > 0 {
			return cached, nil
		}
	}
	c.mu.Lock()
	c.devices = updated
	c.cacheUntil = time.Now().Add(c.cacheTTL)
	devices := publicDevices(c.devices)
	c.mu.Unlock()
	return devices, nil
}

func publicDevices(records map[string]deviceRecord) []domain.CastDevice {
	devices := make([]domain.CastDevice, 0, len(records))
	for _, record := range records {
		devices = append(devices, record.device)
	}
	sort.Slice(devices, func(left, right int) bool {
		return strings.ToLower(devices[left].Name) < strings.ToLower(devices[right].Name)
	})
	return devices
}

func discoverLocations(ctx context.Context, timeout time.Duration) ([]string, error) {
	addresses := discoveryAddresses()
	results := make(chan []string, len(addresses))
	errors := make(chan error, len(addresses))
	var wait sync.WaitGroup
	for _, address := range addresses {
		address := address
		wait.Add(1)
		go func() {
			defer wait.Done()
			locations, err := probeSSDP(ctx, address, timeout)
			if err != nil {
				errors <- err
				return
			}
			results <- locations
		}()
	}
	wait.Wait()
	close(results)
	close(errors)

	unique := make(map[string]struct{})
	for group := range results {
		for _, location := range group {
			unique[location] = struct{}{}
		}
	}
	locations := make([]string, 0, len(unique))
	for location := range unique {
		locations = append(locations, location)
	}
	sort.Strings(locations)
	if len(locations) == 0 {
		for err := range errors {
			return nil, fmt.Errorf("%w: UPnP seadmete otsing ebaõnnestus: %v", domain.ErrUnavailable, err)
		}
	}
	return locations, nil
}

func discoveryAddresses() []net.IP {
	addresses := []net.IP{nil}
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}
	seen := make(map[string]struct{})
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 || networkInterface.Flags&net.FlagMulticast == 0 {
			continue
		}
		interfaceAddresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range interfaceAddresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			key := ip.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			addresses = append(addresses, ip)
		}
	}
	return addresses
}

func probeSSDP(ctx context.Context, localIP net.IP, timeout time.Duration) ([]string, error) {
	connection, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localIP, Port: 0})
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	target, err := net.ResolveUDPAddr("udp4", ssdpAddress)
	if err != nil {
		return nil, err
	}
	for _, searchTarget := range []string{mediaRendererSearchTarget, "ssdp:all"} {
		message := strings.Join([]string{
			"M-SEARCH * HTTP/1.1",
			"HOST: " + ssdpAddress,
			`MAN: "ssdp:discover"`,
			"MX: 1",
			"ST: " + searchTarget,
			"", "",
		}, "\r\n")
		if _, err := connection.WriteToUDP([]byte(message), target); err != nil {
			return nil, err
		}
	}

	deadline := time.Now().Add(timeout)
	locations := make(map[string]struct{})
	buffer := make([]byte, 64<<10)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		readDeadline := time.Now().Add(200 * time.Millisecond)
		if readDeadline.After(deadline) {
			readDeadline = deadline
		}
		_ = connection.SetReadDeadline(readDeadline)
		read, _, err := connection.ReadFromUDP(buffer)
		if err != nil {
			if networkError, ok := err.(net.Error); ok && networkError.Timeout() {
				continue
			}
			return nil, err
		}
		if location := responseHeader(string(buffer[:read]), "location"); location != "" {
			locations[strings.TrimSpace(location)] = struct{}{}
			if len(locations) >= 64 {
				break
			}
		}
	}
	result := make([]string, 0, len(locations))
	for location := range locations {
		result = append(result, location)
	}
	return result, nil
}

func responseHeader(response, name string) string {
	for _, line := range strings.Split(strings.ReplaceAll(response, "\r\n", "\n"), "\n") {
		key, value, found := strings.Cut(line, ":")
		if found && strings.EqualFold(strings.TrimSpace(key), name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (c *Client) describe(ctx context.Context, rawLocation string) (deviceRecord, error) {
	location, err := url.Parse(strings.TrimSpace(rawLocation))
	if err != nil || !isLocalURL(ctx, location) {
		return deviceRecord{}, fmt.Errorf("invalid local UPnP description URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
	if err != nil {
		return deviceRecord{}, err
	}
	request.Header.Set("User-Agent", "KoduMedia/1.0 UPnP/1.1")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return deviceRecord{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return deviceRecord{}, fmt.Errorf("UPnP description returned %s", response.Status)
	}
	var root descriptionRoot
	decoder := xml.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&root); err != nil {
		return deviceRecord{}, err
	}
	renderer := findRenderer(root.Device)
	if renderer == nil {
		return deviceRecord{}, fmt.Errorf("device is not a media renderer")
	}
	base := location
	if strings.TrimSpace(root.URLBase) != "" {
		if parsed, parseErr := url.Parse(strings.TrimSpace(root.URLBase)); parseErr == nil && sameLocalHost(ctx, location, parsed) {
			base = parsed
		}
	}

	var avTransport, renderingControl descriptionService
	for _, service := range renderer.ServiceList {
		switch {
		case strings.HasPrefix(service.ServiceType, avTransportServicePrefix):
			avTransport = service
		case strings.HasPrefix(service.ServiceType, renderingServicePrefix):
			renderingControl = service
		}
	}
	if avTransport.ServiceType == "" {
		return deviceRecord{}, fmt.Errorf("media renderer has no AVTransport service")
	}
	avURL, err := resolveControlURL(ctx, base, avTransport.ControlURL)
	if err != nil {
		return deviceRecord{}, err
	}
	var renderingURL *url.URL
	if renderingControl.ServiceType != "" {
		renderingURL, _ = resolveControlURL(ctx, base, renderingControl.ControlURL)
	}
	model := strings.TrimSpace(renderer.ModelName)
	if model == "" {
		model = strings.TrimSpace(renderer.ModelNumber)
	}
	name := strings.TrimSpace(renderer.FriendlyName)
	if name == "" {
		name = model
	}
	if name == "" {
		name = "DLNA teler"
	}
	identity := strings.TrimSpace(renderer.UDN)
	if identity == "" {
		identity = location.String()
	}
	digest := sha256.Sum256([]byte(identity))
	id := hex.EncodeToString(digest[:10])
	device := domain.CastDevice{
		ID: id, Name: name, Manufacturer: strings.TrimSpace(renderer.Manufacturer), Model: model,
		Protocol: domain.CastProtocolDLNA, CanSeek: true, CanSetVolume: renderingURL != nil,
	}
	return deviceRecord{
		device: device, location: location, avTransportURL: avURL, avTransportType: avTransport.ServiceType,
		renderingControlURL: renderingURL, renderingControlType: renderingControl.ServiceType,
	}, nil
}

func findRenderer(device descriptionDevice) *descriptionDevice {
	if strings.Contains(strings.ToLower(device.DeviceType), "device:mediarenderer:") {
		copy := device
		return &copy
	}
	for _, embedded := range device.EmbeddedDevices {
		if renderer := findRenderer(embedded); renderer != nil {
			return renderer
		}
	}
	return nil
}

func resolveControlURL(ctx context.Context, base *url.URL, rawControlURL string) (*url.URL, error) {
	reference, err := url.Parse(strings.TrimSpace(rawControlURL))
	if err != nil {
		return nil, err
	}
	resolved := base.ResolveReference(reference)
	if !sameLocalHost(ctx, base, resolved) {
		return nil, fmt.Errorf("UPnP control URL left the device host")
	}
	return resolved, nil
}

func isLocalURL(ctx context.Context, value *url.URL) bool {
	if value == nil || (value.Scheme != "http" && value.Scheme != "https") || value.Hostname() == "" {
		return false
	}
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", value.Hostname())
	if err != nil || len(addresses) == 0 {
		return false
	}
	for _, address := range addresses {
		if !(address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast()) {
			return false
		}
	}
	return true
}

func sameLocalHost(ctx context.Context, left, right *url.URL) bool {
	if left == nil || right == nil || !isLocalURL(ctx, right) {
		return false
	}
	return strings.EqualFold(left.Hostname(), right.Hostname())
}

func (c *Client) device(ctx context.Context, id string) (deviceRecord, error) {
	c.mu.RLock()
	record, found := c.devices[id]
	c.mu.RUnlock()
	if found {
		return record, nil
	}
	if _, err := c.Discover(ctx); err != nil {
		return deviceRecord{}, err
	}
	c.mu.RLock()
	record, found = c.devices[id]
	c.mu.RUnlock()
	if !found {
		return deviceRecord{}, fmt.Errorf("%w: castimise seadet ei leitud", domain.ErrNotFound)
	}
	return record, nil
}

func (c *Client) ReachableMediaURL(ctx context.Context, deviceID, mediaURL string) (string, error) {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(mediaURL)
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("%w: vigane castimise meedia-URL", domain.ErrInvalidInput)
	}
	if parsed.Scheme == "https" {
		return parsed.String(), nil
	}
	targetAddresses, err := net.DefaultResolver.LookupIP(ctx, "ip4", record.location.Hostname())
	if err != nil || len(targetAddresses) == 0 {
		return parsed.String(), nil
	}
	connection, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: targetAddresses[0], Port: 9})
	if err != nil {
		return parsed.String(), nil
	}
	localAddress, _ := connection.LocalAddr().(*net.UDPAddr)
	_ = connection.Close()
	if localAddress == nil || localAddress.IP == nil || localAddress.IP.IsUnspecified() {
		return parsed.String(), nil
	}
	if port := parsed.Port(); port != "" {
		parsed.Host = net.JoinHostPort(localAddress.IP.String(), port)
	} else {
		parsed.Host = localAddress.IP.String()
	}
	return parsed.String(), nil
}

func (c *Client) Load(ctx context.Context, deviceID string, request ports.CastLoadRequest) error {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return err
	}
	metadata := didlMetadata(request.MediaURL, request.Metadata)
	if _, err := c.soap(ctx, record.avTransportURL, record.avTransportType, "SetAVTransportURI", []soapArgument{
		{Name: "InstanceID", Value: "0"},
		{Name: "CurrentURI", Value: request.MediaURL},
		{Name: "CurrentURIMetaData", Value: metadata},
	}); err != nil {
		return err
	}
	if err := c.Play(ctx, deviceID); err != nil {
		return err
	}
	if request.PositionMS <= 500 {
		return nil
	}
	var seekErr error
	for attempt := 0; attempt < 4; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(350+attempt*250) * time.Millisecond):
		}
		seekErr = c.Seek(ctx, deviceID, request.PositionMS)
		if seekErr == nil {
			_ = c.Play(ctx, deviceID)
			return nil
		}
	}
	slog.Warn("UPnP renderer started from the beginning because seek was rejected", "device", record.device.Name, "error", seekErr)
	return nil
}

func (c *Client) Play(ctx context.Context, deviceID string) error {
	return c.avTransportAction(ctx, deviceID, "Play", []soapArgument{{Name: "InstanceID", Value: "0"}, {Name: "Speed", Value: "1"}})
}

func (c *Client) Pause(ctx context.Context, deviceID string) error {
	return c.avTransportAction(ctx, deviceID, "Pause", []soapArgument{{Name: "InstanceID", Value: "0"}})
}

func (c *Client) Stop(ctx context.Context, deviceID string) error {
	return c.avTransportAction(ctx, deviceID, "Stop", []soapArgument{{Name: "InstanceID", Value: "0"}})
}

func (c *Client) Seek(ctx context.Context, deviceID string, positionMS int64) error {
	if positionMS < 0 {
		positionMS = 0
	}
	return c.avTransportAction(ctx, deviceID, "Seek", []soapArgument{
		{Name: "InstanceID", Value: "0"},
		{Name: "Unit", Value: "REL_TIME"},
		{Name: "Target", Value: formatUPnPTime(positionMS)},
	})
}

func (c *Client) avTransportAction(ctx context.Context, deviceID, action string, arguments []soapArgument) error {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return err
	}
	_, err = c.soap(ctx, record.avTransportURL, record.avTransportType, action, arguments)
	return err
}

func (c *Client) SetVolume(ctx context.Context, deviceID string, volume float64) error {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return err
	}
	if record.renderingControlURL == nil {
		return fmt.Errorf("%w: seade ei luba KoduMedial helitugevust muuta", domain.ErrUnsupported)
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	_, err = c.soap(ctx, record.renderingControlURL, record.renderingControlType, "SetVolume", []soapArgument{
		{Name: "InstanceID", Value: "0"}, {Name: "Channel", Value: "Master"},
		{Name: "DesiredVolume", Value: strconv.Itoa(int(volume*100 + 0.5))},
	})
	return err
}

func (c *Client) SetMuted(ctx context.Context, deviceID string, muted bool) error {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return err
	}
	if record.renderingControlURL == nil {
		return fmt.Errorf("%w: seade ei luba KoduMedial heli vaigistada", domain.ErrUnsupported)
	}
	value := "0"
	if muted {
		value = "1"
	}
	_, err = c.soap(ctx, record.renderingControlURL, record.renderingControlType, "SetMute", []soapArgument{
		{Name: "InstanceID", Value: "0"}, {Name: "Channel", Value: "Master"}, {Name: "DesiredMute", Value: value},
	})
	return err
}

func (c *Client) Status(ctx context.Context, deviceID string) (domain.CastDeviceStatus, error) {
	record, err := c.device(ctx, deviceID)
	if err != nil {
		return domain.CastDeviceStatus{}, err
	}
	transportBody, err := c.soap(ctx, record.avTransportURL, record.avTransportType, "GetTransportInfo", []soapArgument{{Name: "InstanceID", Value: "0"}})
	if err != nil {
		return domain.CastDeviceStatus{}, err
	}
	transportValues := soapValues(transportBody)
	status := domain.CastDeviceStatus{State: normalizeTransportState(transportValues["CurrentTransportState"])}
	if positionBody, positionErr := c.soap(ctx, record.avTransportURL, record.avTransportType, "GetPositionInfo", []soapArgument{{Name: "InstanceID", Value: "0"}}); positionErr == nil {
		values := soapValues(positionBody)
		status.PositionMS = parseUPnPTime(values["RelTime"])
		status.DurationMS = parseUPnPTime(values["TrackDuration"])
	}
	if record.renderingControlURL != nil {
		if volumeBody, volumeErr := c.soap(ctx, record.renderingControlURL, record.renderingControlType, "GetVolume", []soapArgument{{Name: "InstanceID", Value: "0"}, {Name: "Channel", Value: "Master"}}); volumeErr == nil {
			if volume, parseErr := strconv.Atoi(soapValues(volumeBody)["CurrentVolume"]); parseErr == nil {
				status.Volume = float64(volume) / 100
			}
		}
		if muteBody, muteErr := c.soap(ctx, record.renderingControlURL, record.renderingControlType, "GetMute", []soapArgument{{Name: "InstanceID", Value: "0"}, {Name: "Channel", Value: "Master"}}); muteErr == nil {
			value := strings.TrimSpace(soapValues(muteBody)["CurrentMute"])
			status.Muted = value == "1" || strings.EqualFold(value, "true")
		}
	}
	return status, nil
}

type soapArgument struct {
	Name  string
	Value string
}

func (c *Client) soap(ctx context.Context, controlURL *url.URL, serviceType, action string, arguments []soapArgument) ([]byte, error) {
	if controlURL == nil || serviceType == "" {
		return nil, fmt.Errorf("%w: UPnP juhtimisteenus puudub", domain.ErrUnsupported)
	}
	var actionBody strings.Builder
	for _, argument := range arguments {
		actionBody.WriteByte('<')
		actionBody.WriteString(argument.Name)
		actionBody.WriteByte('>')
		var escaped bytes.Buffer
		_ = xml.EscapeText(&escaped, []byte(argument.Value))
		actionBody.Write(escaped.Bytes())
		actionBody.WriteString("</")
		actionBody.WriteString(argument.Name)
		actionBody.WriteByte('>')
	}
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body><u:` + action + ` xmlns:u="` + serviceType + `">` + actionBody.String() + `</u:` + action + `></s:Body></s:Envelope>`
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	request.Header.Set("SOAPACTION", `"`+serviceType+`#`+action+`"`)
	request.Header.Set("User-Agent", "KoduMedia/1.0 UPnP/1.1")
	request.Close = true
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: teleriga ei saanud ühendust: %v", domain.ErrUnavailable, err)
	}
	defer response.Body.Close()
	payload, readErr := io.ReadAll(io.LimitReader(response.Body, 256<<10))
	if readErr != nil {
		return nil, fmt.Errorf("%w: teleri vastust ei saanud lugeda: %v", domain.ErrUnavailable, readErr)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		fault := soapValues(payload)["errorDescription"]
		if fault == "" {
			fault = response.Status
		}
		return nil, fmt.Errorf("%w: teler keeldus käsust %s (%s)", domain.ErrUnavailable, action, fault)
	}
	return payload, nil
}

func soapValues(payload []byte) map[string]string {
	values := make(map[string]string)
	decoder := xml.NewDecoder(bytes.NewReader(payload))
	stack := make([]string, 0, 8)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch value := token.(type) {
		case xml.StartElement:
			stack = append(stack, value.Name.Local)
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(value))
			if text != "" {
				values[stack[len(stack)-1]] = text
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return values
}

func didlMetadata(mediaURL string, metadata domain.CastMediaMetadata) string {
	contentType := strings.TrimSpace(metadata.ContentType)
	if contentType == "" {
		contentType = "video/mp4"
	}
	var escapedURL, escapedTitle, escapedSubtitle bytes.Buffer
	_ = xml.EscapeText(&escapedURL, []byte(mediaURL))
	_ = xml.EscapeText(&escapedTitle, []byte(metadata.Title))
	_ = xml.EscapeText(&escapedSubtitle, []byte(metadata.Subtitle))
	return `<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/">` +
		`<item id="0" parentID="-1" restricted="1"><dc:title>` + escapedTitle.String() + `</dc:title>` +
		`<dc:description>` + escapedSubtitle.String() + `</dc:description><upnp:class>object.item.videoItem</upnp:class>` +
		`<res protocolInfo="http-get:*:` + contentType + `:*">` + escapedURL.String() + `</res></item></DIDL-Lite>`
}

func normalizeTransportState(value string) domain.CastPlaybackState {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PLAYING":
		return domain.CastStatePlaying
	case "PAUSED_PLAYBACK", "PAUSED_RECORDING":
		return domain.CastStatePaused
	case "TRANSITIONING":
		return domain.CastStateBuffering
	case "STOPPED", "NO_MEDIA_PRESENT":
		return domain.CastStateStopped
	default:
		return domain.CastStateUnknown
	}
}

func formatUPnPTime(milliseconds int64) string {
	if milliseconds < 0 {
		milliseconds = 0
	}
	totalSeconds := milliseconds / 1000
	hours := totalSeconds / 3600
	minutes := totalSeconds % 3600 / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func parseUPnPTime(value string) int64 {
	value = strings.TrimSpace(strings.Split(value, ".")[0])
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0
	}
	hours, hourErr := strconv.ParseInt(parts[0], 10, 64)
	minutes, minuteErr := strconv.ParseInt(parts[1], 10, 64)
	seconds, secondErr := strconv.ParseInt(parts[2], 10, 64)
	if hourErr != nil || minuteErr != nil || secondErr != nil || hours < 0 || minutes < 0 || seconds < 0 {
		return 0
	}
	return ((hours*60+minutes)*60 + seconds) * 1000
}
