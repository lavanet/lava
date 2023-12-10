package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
)

const (
	CacheMaxCost               = 10 * 1024 // 10K cost
	CacheNumCounters           = 100000    // expect 10K items
	FrozenProviderAttribute    = "frozen_provider_alert"
	SubscriptionAlertAttribute = "subscription_limit_alert"
	UnhealthyProviderAttribute = "unhealthy_provider_alert"
	UnhealthyConsumerAttribute = "unhealthy_consumer_alert"
	ProviderBlockGapAttribute  = "provider_block_gap_alert"
	ConsumerBlockGapAttribute  = "consumer_block_gap_alert"
	ProviderLatencyAttribute   = "provider_latency_alert"
	red                        = "#ff0000"
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
	SendSameAlertWithoutInterval  bool
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
}

func NewAlerting(options AlertingOptions) *Alerting {
	al := &Alerting{}
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
	if !options.SendSameAlertWithoutInterval {
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
	} else {
		al.sameAlertInterval = 0
	}
	return al
}

func (al *Alerting) SendUrlAlert(alert string, attrs []utils.Attribute) error {
	attachments := []map[string]interface{}{}
	fields := []map[string]interface{}{}
	for _, attr := range attrs {
		attrMap := map[string]interface{}{
			"title":  attr.Key,
			"name":   attr.Key,
			"value":  utils.StrValue(attr.Value),
			"short":  false,
			"inline": true,
		}
		fields = append(fields, attrMap)
	}
	attachment := map[string]interface{}{
		"text":   "Data",
		"color":  red,
		"fields": fields,
	}
	attachments = append(attachments, attachment)
	payload := map[string]interface{}{
		"text":        alert,
		"title":       alert,
		"attachments": attachments,
		"embeds":      attachments,
	}
	payloadBytes, err := json.Marshal(payload)
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

func (al *Alerting) SendAlert(alert string, attrs []utils.Attribute) {
	if al.identifier != "" {
		attrs = append(attrs, utils.LogAttr("identifier", al.identifier))
	}

	if al.sameAlertInterval > 0 && al.AlertsCache != nil {
		slices.SortStableFunc(attrs, func(attr1, attr2 utils.Attribute) bool {
			return attr1.Key < attr2.Key
		})
		hashStr := string(sigs.HashMsg([]byte(fmt.Sprintf("%s %v", alert, attrs))))
		storedVal, found := al.AlertsCache.Get(hashStr)
		if found {
			// was already in the cache
			storedTime, ok := storedVal.(time.Time)
			if !ok {
				utils.LavaFormatFatal("invalid usage of cache", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
			}
			if !time.Now().After(storedTime.Add(al.sameAlertInterval)) {
				// filter this alert
				return
			}
		}
		al.AlertsCache.SetWithTTL(hashStr, time.Now(), 1, al.sameAlertInterval)
	}

	if al.logging {
		utils.LavaFormatError(alert, nil, attrs...)
	}
	if al.url != "" {
		al.SendUrlAlert(alert, attrs)
	}
}

func (al *Alerting) SendFrozenProviders(frozenProviders map[LavaEntity]struct{}) {
	providers := map[string][]string{}
	attrs := []utils.Attribute{}
	for frozen := range frozenProviders {
		providers[frozen.Address] = append(providers[frozen.Address], frozen.SpecId)
	}
	for providerAddr, chains := range providers {
		attrs = append(attrs, utils.LogAttr(providerAddr, strings.Join(chains, ",")))
	}
	if len(attrs) > 0 {
		al.SendAlert(FrozenProviderAttribute, attrs)
	}
}

func (al *Alerting) UnhealthyProviders(unhealthy map[LavaEntity]string) {
	attrs := []utils.Attribute{}
	for provider, errSt := range unhealthy {
		attrs = append(attrs, utils.LogAttr(provider.Address, fmt.Sprintf("spec: %s, err: %s", provider.SpecId, errSt)))
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
	attrs := []utils.Attribute{}
	for subscriptionAddr, data := range subs {
		if reason, alert := al.ShouldAlertSubscription(data); alert {
			attrs = append(attrs, utils.LogAttr(subscriptionAddr, reason))
		}
	}
	if len(attrs) > 0 {
		al.SendAlert(SubscriptionAlertAttribute, attrs)
	}
}

func (al *Alerting) ProvidersAlerts(healthResults *HealthResults) {
	attrs := []utils.Attribute{}
	attrsForLatency := []utils.Attribute{}
	for provider, data := range healthResults.ProviderData {
		specId := provider.SpecId
		if al.allowedTimeGapVsReference > 0 {
			latestBlock := healthResults.LatestBlocks[specId]
			if latestBlock > data.block {
				gap := latestBlock - data.block
				timeGap := time.Duration(gap*healthResults.Specs[specId].AverageBlockTime) * time.Millisecond
				if timeGap > al.allowedTimeGapVsReference {
					attrs = append(attrs, utils.LogAttr(provider.Address, fmt.Sprintf("spec: %s, block gap: %s/%s", provider.SpecId, utils.StrValue(data.block), utils.StrValue(latestBlock))))
				}
			}
		}
		if al.maxProviderLatency > 0 {
			if data.latency > al.maxProviderLatency {
				attrsForLatency = append(attrsForLatency, utils.LogAttr(provider.Address, fmt.Sprintf("spec: %s, latency: %s/%s", provider.SpecId, utils.StrValue(data.latency), utils.StrValue(al.maxProviderLatency))))
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
	attrs := []utils.Attribute{}
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
					attrs = append(attrs, utils.LogAttr(consumer.Address, fmt.Sprintf("spec: %s, block gap: %s/%s", consumer.SpecId, utils.StrValue(consumerBlock), utils.StrValue(latestBlock))))
				}
			}
		}
	}
	if len(attrs) > 0 {
		al.SendAlert(ConsumerBlockGapAttribute, attrs)
	}
	attrsUnhealthy := []utils.Attribute{}
	for consumer, errSt := range healthResults.UnhealthyConsumers {
		attrsUnhealthy = append(attrsUnhealthy, utils.LogAttr(consumer.Address, fmt.Sprintf("spec: %s, err: %s", consumer.SpecId, errSt)))
	}
	if len(attrsUnhealthy) > 0 {
		al.SendAlert(UnhealthyConsumerAttribute, attrsUnhealthy)
	}
}

func (al *Alerting) CheckHealthResults(healthResults *HealthResults) {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()

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
}
