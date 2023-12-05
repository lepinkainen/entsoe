package main

import (
	"encoding/xml"
	"time"
)

type PublicationMarketDocument struct {
	XMLName                   xml.Name          `xml:"Publication_MarketDocument"`
	MRID                      string            `xml:"mRID"`
	RevisionNumber            int               `xml:"revisionNumber"`
	Type                      string            `xml:"type"`
	SenderMarketParticipant   MarketParticipant `xml:"sender_MarketParticipant"`
	ReceiverMarketParticipant MarketParticipant `xml:"receiver_MarketParticipant"`
	CreatedDateTime           time.Time         `xml:"createdDateTime"`
	PeriodTimeInterval        TimeInterval      `xml:"period.timeInterval"`
	TimeSeries                []TimeSeries      `xml:"TimeSeries"`
}

type AcknowledgementMarketDocument struct {
	XMLName                               xml.Name  `xml:"Acknowledgement_MarketDocument"`
	MRID                                  string    `xml:"mRID"`
	CreatedDateTime                       time.Time `xml:"createdDateTime"`
	SenderMarketParticipantMRID           string    `xml:"sender_MarketParticipant>mRID"`
	SenderMarketParticipantType           string    `xml:"sender_MarketParticipant>marketRole>type"`
	ReceiverMarketParticipantMRID         string    `xml:"receiver_MarketParticipant>mRID"`
	ReceiverMarketParticipantType         string    `xml:"receiver_MarketParticipant>marketRole>type"`
	ReceivedMarketDocumentCreatedDateTime time.Time `xml:"received_MarketDocument>createdDateTime"`
	Reason                                struct {
		Code string `xml:"code"`
		Text string `xml:"text"`
	} `xml:"Reason"`
}

type MarketParticipant struct {
	MRID         string `xml:"mRID,attr"`
	CodingScheme string `xml:"codingScheme,attr"`
	MarketRole   struct {
		Type string `xml:"type"`
	} `xml:"marketRole"`
}

type TimeInterval struct {
	Start string `xml:"start"`
	End   string `xml:"end"`
}

type TimeSeries struct {
	MRID         string `xml:"mRID"`
	BusinessType string `xml:"businessType"`
	InDomainMRID struct {
		CodingScheme string `xml:"codingScheme,attr"`
		MRID         string `xml:",chardata"`
	} `xml:"in_Domain.mRID"`
	OutDomainMRID struct {
		CodingScheme string `xml:"codingScheme,attr"`
		MRID         string `xml:",chardata"`
	} `xml:"out_Domain.mRID"`
	CurrencyUnitName     string `xml:"currency_Unit.name"`
	PriceMeasureUnitName string `xml:"price_Measure_Unit.name"`
	CurveType            string `xml:"curveType"`
	Period               struct {
		TimeInterval TimeInterval `xml:"timeInterval"`
		Resolution   string       `xml:"resolution"`
		Points       []Point      `xml:"Point"`
	} `xml:"Period"`
}

type Point struct {
	Position    int     `xml:"position"`
	PriceAmount float64 `xml:"price.amount"`
}
