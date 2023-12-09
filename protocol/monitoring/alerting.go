package monitoring

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lavanet/lava/utils"
)

const (
	FrozenProviderAttrbiute    = "frozen_provider"
	SubscriptionAlertAttribute = "subscription"
	red                        = "#ff0000"
	green                      = "#00ff00"
	UsagePercentageAlert       = "percentage of cu too low"
	LeftTimeAlert              = "left subscription time is too low"
	MonthDuration              = 30 * 24 * time.Hour
)

type AlertingOptions struct {
	Url                           string
	Logging                       bool
	Identifier                    string
	SubscriptionCUPercentageAlert float64
	SubscriptionLeftTimeAlert     time.Duration
}

type Alerting struct {
	url                           string
	logging                       bool
	identifier                    string
	subscriptionCUPercentageAlert float64
	subscriptionLeftTimeAlert     time.Duration
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
		al.SendAlert(FrozenProviderAttrbiute, attrs)
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

func (al *Alerting) CheckHealthResults(healthResults *HealthResults) {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	// handle frozen providers
	frozenProviders := healthResults.FrozenProviders
	if len(frozenProviders) > 0 {
		al.SendFrozenProviders(frozenProviders)
	}
	// handle subscriptions
	al.CheckSubscriptionData(healthResults.SubscriptionsData)
}
