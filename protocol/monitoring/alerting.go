package monitoring

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/sigs"
)

const (
	CacheMaxCost               = 10 * 1024 // 10K cost
	CacheNumCounters           = 100000    // expect 10K items
	OKString                   = "OK"
	FrozenProviderAttribute    = "frozen_provider_alert"
	SubscriptionAlertAttribute = "subscription_limit_alert"
	UnhealthyProviderAttribute = "unhealthy_provider_alert"
	UnhealthyConsumerAttribute = "unhealthy_consumer_alert"
	ProviderBlockGapAttribute  = "provider_block_gap_alert"
	ConsumerBlockGapAttribute  = "consumer_block_gap_alert"
	ProviderLatencyAttribute   = "provider_latency_alert"
	red                        = "#ff0000"
	lessRed                    = "#990000"
	green                      = "#00ff00"
	UsagePercentageAlert       = "percentage of cu too low"
	LeftTimeAlert              = "left subscription time is too low"
	MonthDuration              = 30 * 24 * time.Hour
	defaultSameAlertInterval   = 6 * time.Hour
)

type AlertingOptions struct {
	Url                           string // where to send the alerts
	Logging                       bool   // wether to log alerts to stdout
	Identifier                    string // a unique identifier added to all alerts
	SubscriptionCUPercentageAlert float64
	SubscriptionLeftTimeAlert     time.Duration
	AllowedTimeGapVsReference     time.Duration
	MaxProviderLatency            time.Duration
	SameAlertInterval             time.Duration
	DisableAlertSuppression       bool
	SuppressionCounterThreshold   uint64
}

type AlertAttribute struct {
	entity LavaEntity
	data   string
}

type AlertEntry struct {
	alertType string
	entity    LavaEntity
}

type AlertCount struct {
	active   uint64
	recovery uint64
}

type Alerting struct {
	url                           string
	logging                       bool
	identifier                    string
	subscriptionCUPercentageAlert float64
	subscriptionLeftTimeAlert     time.Duration
	allowedTimeGapVsReference     time.Duration
	maxProviderLatency            time.Duration
	sameAlertInterval             time.Duration
	AlertsCache                   *ristretto.Cache
	activeAlerts                  map[AlertEntry]AlertCount // count how many occurrences of an alert
	healthy                       map[LavaEntity]struct{}
	unhealthy                     map[LavaEntity]struct{}
	currentAlerts                 map[AlertEntry]struct{}
	suppressionCounterThreshold   uint64
	suppressedAlerts              uint64 // monitoring
	payload                       map[string]interface{}
	colorToggle                   bool
}

func NewAlerting(options AlertingOptions) *Alerting {
	al := &Alerting{
		activeAlerts:  map[AlertEntry]AlertCount{},
		healthy:       map[LavaEntity]struct{}{},
		unhealthy:     map[LavaEntity]struct{}{},
		currentAlerts: map[AlertEntry]struct{}{},
		payload:       map[string]interface{}{},
	}
	if options.Url != "" {
		al.url = options.Url
	}
	if options.Identifier != "" {
		al.identifier = options.Identifier
	}
	if options.Logging {
		al.logging = true
	}
	al.subscriptionCUPercentageAlert = options.SubscriptionCUPercentageAlert
	al.subscriptionLeftTimeAlert = options.SubscriptionLeftTimeAlert
	al.allowedTimeGapVsReference = options.AllowedTimeGapVsReference
	al.maxProviderLatency = options.MaxProviderLatency
	al.suppressionCounterThreshold = options.SuppressionCounterThreshold
	if options.DisableAlertSuppression {
		al.sameAlertInterval = 0
		al.suppressionCounterThreshold = 0
	} else {
		if options.SameAlertInterval != 0 {
			al.sameAlertInterval = options.SameAlertInterval
		} else {
			al.sameAlertInterval = defaultSameAlertInterval
		}
		cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
		if err != nil {
			utils.LavaFormatFatal("failed setting up cache for queries", err)
		}
		al.AlertsCache = cache
	}
	return al
}

func (al *Alerting) FilterOccurenceSuppresedAlerts(alert string, attributes []AlertAttribute) (filteredAttributes []AlertAttribute) {
	if al.suppressionCounterThreshold <= 1 {
		return attributes
	}
	for _, attr := range attributes {
		alertEntity := AlertEntry{
			alertType: alert,
			entity:    attr.entity,
		}
		alertCount := al.activeAlerts[alertEntity]
		alertCount.active++
		al.currentAlerts[alertEntity] = struct{}{} // so we can clear keys that weren't changed

		al.activeAlerts[alertEntity] = alertCount
		if alertCount.active >= al.suppressionCounterThreshold {
			filteredAttributes = append(filteredAttributes, attr)
			continue
		}
		al.suppressedAlerts++
	}
	return filteredAttributes
}

// func (al *Alerting) SendAlert(alert string, attrs []utils.Attribute) {
func (al *Alerting) SendAlert(alert string, attributes []AlertAttribute) {
	// check for occurrence suppression
	attributes = al.FilterOccurenceSuppresedAlerts(alert, attributes)
	if len(attributes) == 0 {
		return
	}
	attrs := al.FilterTimeSuppresedAlerts(attributes, alert)
	if len(attrs) == 0 {
		return
	}

	if al.url != "" {
		go al.AppendUrlAlert(alert, attrs)
	}
	if al.logging {
		if al.identifier != "" {
			alert = alert + " - " + al.identifier
		}
		utils.LavaFormatError(alert, nil, attrs...)
	}
}

func (al *Alerting) FilterTimeSuppresedAlerts(attributes []AlertAttribute, alert string) []utils.Attribute {
	attrs := []utils.Attribute{}
	for _, attr := range attributes {
		if al.sameAlertInterval > 0 && al.AlertsCache != nil {
			// we only hash by keys, values can differ (like blocks or error)
			hashStr := string(sigs.HashMsg([]byte(fmt.Sprintf("%s %s", alert, attr.entity.String()))))
			storedVal, found := al.AlertsCache.Get(hashStr)
			if found {
				// was already in the cache
				storedTime, ok := storedVal.(time.Time)
				if !ok {
					utils.LavaFormatFatal("invalid usage of cache", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
				}
				if !time.Now().After(storedTime.Add(al.sameAlertInterval)) {
					// filter this alert
					al.suppressedAlerts++
					continue
				}
			}
			al.AlertsCache.SetWithTTL(hashStr, time.Now(), 1, al.sameAlertInterval)
		}
		attrs = append(attrs, utils.LogAttr(attr.entity.String(), attr.data))
	}
	return attrs
}

func (al *Alerting) SendRecoveryAlerts(alertEntries []AlertEntry) {
	alertTypeAttributes := map[string][]utils.Attribute{}
	for _, alertEntry := range alertEntries {
		count, ok := al.activeAlerts[alertEntry]
		if !ok {
			continue
		}
		if count.active < al.suppressionCounterThreshold {
			continue
		}
		attrs, ok := alertTypeAttributes[alertEntry.alertType]
		if !ok {
			attrs = []utils.Attribute{}
		}
		attrs = append(attrs, utils.Attribute{
			Key:   alertEntry.entity.String(),
			Value: OKString,
		})
		alertTypeAttributes[alertEntry.alertType] = attrs
	}
	if len(alertTypeAttributes) == 0 {
		return
	}
	for alertType, attrs := range alertTypeAttributes {
		alertType = "recovered - " + alertType
		if al.url != "" {
			al.AppendUrlAlert(alertType, attrs)
		}
		if al.logging {
			utils.LavaFormatInfo(alertType, attrs...)
		}
	}
}

func (al *Alerting) SendAppendedAlerts() error {
	if len(al.payload) == 0 {
		return nil
	}
	if al.identifier != "" {
		al.payload["text"] = al.identifier
		al.payload["content"] = al.identifier
	}
	payloadBytes, err := json.Marshal(al.payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", al.url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (al *Alerting) AppendUrlAlert(alert string, attrs []utils.Attribute) {
	attachments := []map[string]interface{}{}
	if attachmentsProp, ok := al.payload["attachments"]; ok {
		attachmentsCasted, ok := attachmentsProp.([]map[string]interface{})
		if ok {
			attachments = attachmentsCasted
		}
	}
	fields := []map[string]interface{}{}
	colorToSet := green
	for _, attr := range attrs {
		if utils.StrValue(attr.Value) != OKString {
			colorToSet = red
			if al.colorToggle {
				al.colorToggle = !al.colorToggle
				colorToSet = lessRed
			}
		}
		field := map[string]interface{}{
			"title":  attr.Key,
			"text":   attr.Key,
			"value":  attr.Value,
			"short":  false,
			"inline": false,
		}
		fields = append(fields, field)
	}
	attachment := map[string]interface{}{
		"text":   alert,
		"title":  alert,
		"color":  colorToSet,
		"fields": fields,
	}
	attachments = append(attachments, attachment)
	al.payload["attachments"] = attachments
	al.payload["embeds"] = attachments
}

func (al *Alerting) SendFrozenProviders(frozenProviders map[LavaEntity]struct{}) {
	providers := map[string][]string{}
	attrs := []AlertAttribute{}
	for frozen := range frozenProviders {
		attrs = append(attrs, AlertAttribute{entity: frozen, data: "frozen"})
		providers[frozen.Address] = append(providers[frozen.Address], frozen.SpecId)
	}
	if len(attrs) > 0 {
		al.SendAlert(FrozenProviderAttribute, attrs)
	}
}

func (al *Alerting) UnhealthyProviders(unhealthy map[LavaEntity]string) {
	attrs := []AlertAttribute{}
	for provider, errSt := range unhealthy {
		attrs = append(attrs, AlertAttribute{entity: provider, data: errSt})
	}
	if len(attrs) > 0 {
		al.SendAlert(UnhealthyProviderAttribute, attrs)
	}
}

func (al *Alerting) ShouldAlertSubscription(data SubscriptionData) (reason string, alert bool) {
	reasons := []string{}
	if al.subscriptionCUPercentageAlert > 0 {
		if data.UsagePercentageLeftThisMonth < al.subscriptionCUPercentageAlert {
			alert = true
			reasons = append(reasons, UsagePercentageAlert+" "+strconv.FormatFloat(data.UsagePercentageLeftThisMonth, 'f', 2, 64)+"/"+strconv.FormatFloat(al.subscriptionCUPercentageAlert, 'f', 2, 64))
		}
	}
	if al.subscriptionLeftTimeAlert != 0 {
		timeLeft := data.DurationLeft + MonthDuration*time.Duration(data.FullMonthsLeft)
		if timeLeft < al.subscriptionLeftTimeAlert {
			alert = true
			reasons = append(reasons, LeftTimeAlert+" "+timeLeft.String()+"/"+al.subscriptionLeftTimeAlert.String())
		}
	}
	reason = strings.Join(reasons, " & ")
	return reason, alert
}

func (al *Alerting) CheckSubscriptionData(subs map[string]SubscriptionData) {
	attrs := []AlertAttribute{}
	for subscriptionAddr, data := range subs {
		if reason, alert := al.ShouldAlertSubscription(data); alert {
			attrs = append(attrs, AlertAttribute{entity: LavaEntity{Address: subscriptionAddr, SpecId: ""}, data: reason})
		}
	}
	if len(attrs) > 0 {
		al.SendAlert(SubscriptionAlertAttribute, attrs)
	}
}

func (al *Alerting) ProvidersAlerts(healthResults *HealthResults) {
	attrs := []AlertAttribute{}
	attrsForLatency := []AlertAttribute{}
	for provider, data := range healthResults.ProviderData {
		specId := provider.SpecId
		if al.allowedTimeGapVsReference > 0 {
			latestBlock := healthResults.LatestBlocks[specId]
			if latestBlock > data.Block {
				gap := latestBlock - data.Block
				specHealthResult, ok := healthResults.Specs[specId]
				if !ok {
					utils.LavaFormatFatal("Invalid specid - missing in healthResults", nil, utils.Attribute{Key: "specId", Value: specId})
				}
				timeGap := time.Duration(gap*specHealthResult.AverageBlockTime) * time.Millisecond
				if timeGap > al.allowedTimeGapVsReference {
					attrs = append(attrs, AlertAttribute{entity: provider, data: fmt.Sprintf("block gap: %s/%s", utils.StrValue(data.Block), utils.StrValue(latestBlock))})
				}
			}
		}
		if al.maxProviderLatency > 0 {
			if data.Latency > al.maxProviderLatency {
				attrsForLatency = append(attrsForLatency, AlertAttribute{entity: provider, data: fmt.Sprintf("latency: %s/%s", utils.StrValue(data.Latency), utils.StrValue(al.maxProviderLatency))})
			}
		}
	}
	if len(attrs) > 0 {
		al.SendAlert(ProviderBlockGapAttribute, attrs)
	}
	if len(attrsForLatency) > 0 {
		al.SendAlert(ProviderLatencyAttribute, attrsForLatency)
	}
}

func (al *Alerting) ConsumerAlerts(healthResults *HealthResults) {
	attrs := []AlertAttribute{}
	for consumer, consumerBlock := range healthResults.ConsumerBlocks {
		specId := consumer.SpecId
		if consumerBlock == 0 {
			// skip these, they are handled in unhealthyConsumers
			continue
		} else if al.allowedTimeGapVsReference > 0 {
			latestBlock := healthResults.LatestBlocks[specId]
			if latestBlock > consumerBlock {
				gap := latestBlock - consumerBlock
				timeGap := time.Duration(gap*healthResults.Specs[specId].AverageBlockTime) * time.Millisecond
				if timeGap > al.allowedTimeGapVsReference {
					attrs = append(attrs, AlertAttribute{entity: consumer, data: fmt.Sprintf("block gap: %s/%s", utils.StrValue(consumerBlock), utils.StrValue(latestBlock))})
				}
			}
		}
	}
	if len(attrs) > 0 {
		al.SendAlert(ConsumerBlockGapAttribute, attrs)
	}
	attrsUnhealthy := []AlertAttribute{}
	for consumer, errSt := range healthResults.UnhealthyConsumers {
		attrsUnhealthy = append(attrsUnhealthy, AlertAttribute{entity: consumer, data: errSt})
	}
	if len(attrsUnhealthy) > 0 {
		al.SendAlert(UnhealthyConsumerAttribute, attrsUnhealthy)
	}
}

func (al *Alerting) CheckHealthResults(healthResults *HealthResults) {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	al.payload = map[string]interface{}{}
	suppressed := al.suppressedAlerts
	// reset healthy
	al.currentAlerts = map[AlertEntry]struct{}{}

	// handle frozen providers
	if len(healthResults.FrozenProviders) > 0 {
		al.SendFrozenProviders(healthResults.FrozenProviders)
	}

	// handle subscriptions
	al.CheckSubscriptionData(healthResults.SubscriptionsData)

	// unhealthy providers
	if len(healthResults.UnhealthyProviders) > 0 {
		al.UnhealthyProviders(healthResults.UnhealthyProviders)
	}

	// check providers latestBlock vs reference
	al.ProvidersAlerts(healthResults)

	// check consumers vs reference
	al.ConsumerAlerts(healthResults)

	// delete alerts that are not active, reset recovery for those that are
	keysToDelete := []AlertEntry{}
	for alertEntry := range al.activeAlerts {
		_, ok := al.currentAlerts[alertEntry]
		if !ok {
			// this entry wasn't alerted currently therefore we can shut it off
			count := al.activeAlerts[alertEntry]
			count.recovery++ // increase recovery
			if count.recovery >= al.suppressionCounterThreshold {
				keysToDelete = append(keysToDelete, alertEntry)
			} else if count.active < al.suppressionCounterThreshold {
				// if the threshold for an alert wasn't reached we suppress alerting too
				count.active = 0
			}

			al.activeAlerts[alertEntry] = count
		} else {
			count := al.activeAlerts[alertEntry]
			count.recovery = 0
			al.activeAlerts[alertEntry] = count
		}
	}
	al.SendRecoveryAlerts(keysToDelete)
	for _, keyToDelete := range keysToDelete {
		delete(al.activeAlerts, keyToDelete)
	}
	al.healthy = map[LavaEntity]struct{}{}
	al.unhealthy = map[LavaEntity]struct{}{}

	for entity := range al.currentAlerts {
		al.unhealthy[entity.entity] = struct{}{}
	}

	allEntities := healthResults.GetAllEntities()
	for entity := range allEntities {
		_, ok := al.unhealthy[entity]
		if !ok {
			// not unhealthy sets healthy
			al.healthy[entity] = struct{}{}
		}
	}
	if len(al.currentAlerts) == 0 {
		utils.LavaFormatInfo("[+] healthy - no new alerts", utils.LogAttr("healthy", uint64(len(al.healthy))))
	} else {
		utils.LavaFormatInfo("[-] unhealthy", utils.LogAttr("healthy", uint64(len(al.healthy))), utils.LogAttr("count", uint64(len(al.unhealthy))), utils.LogAttr("currently suppressed", al.suppressedAlerts-suppressed))
	}
	al.SendAppendedAlerts()
}

func (al *Alerting) ActiveAlerts() (alerts uint64, unhealthy uint64, healthy uint64) {
	return uint64(len(al.activeAlerts)), uint64(len(al.unhealthy)), uint64(len(al.healthy))
}
