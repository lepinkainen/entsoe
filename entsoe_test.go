//go:build !ci

package main

import (
	"encoding/xml"
	"testing"
)

func TestPublicationMarketDocumentUnmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Publication_MarketDocument xmlns="urn:iec62325.351:tc57wg16:451-3:publicationdocument:7:3">
	<mRID>test-id</mRID>
	<revisionNumber>1</revisionNumber>
	<type>A44</type>
	<sender_MarketParticipant.mRID codingScheme="A01">10X1001A1001A450</sender_MarketParticipant.mRID>
	<sender_MarketParticipant.marketRole.type>A32</sender_MarketParticipant.marketRole.type>
	<receiver_MarketParticipant.mRID codingScheme="A01">10X1001A1001A450</receiver_MarketParticipant.mRID>
	<receiver_MarketParticipant.marketRole.type>A33</receiver_MarketParticipant.marketRole.type>
	<createdDateTime>2023-01-01T00:00:00Z</createdDateTime>
	<period.timeInterval>
		<start>2023-01-01T00:00Z</start>
		<end>2023-01-02T00:00Z</end>
	</period.timeInterval>
	<TimeSeries>
		<mRID>1</mRID>
		<businessType>A62</businessType>
		<objectAggregation>A01</objectAggregation>
		<in_Domain.mRID codingScheme="A01">10YFI-1--------U</in_Domain.mRID>
		<out_Domain.mRID codingScheme="A01">10YFI-1--------U</out_Domain.mRID>
		<currency_Unit.name>EUR</currency_Unit.name>
		<price_Measure_Unit.name>MWH</price_Measure_Unit.name>
		<curveType>A01</curveType>
		<Period>
			<timeInterval>
				<start>2023-01-01T00:00Z</start>
				<end>2023-01-02T00:00Z</end>
			</timeInterval>
			<resolution>PT60M</resolution>
			<Point>
				<position>1</position>
				<price.amount>50.00</price.amount>
			</Point>
		</Period>
	</TimeSeries>
</Publication_MarketDocument>`

	var doc PublicationMarketDocument
	err := xml.Unmarshal([]byte(xmlData), &doc)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if doc.MRID != "test-id" {
		t.Errorf("Expected mRID 'test-id', got '%s'", doc.MRID)
	}

	if len(doc.TimeSeries) != 1 {
		t.Errorf("Expected 1 TimeSeries, got %d", len(doc.TimeSeries))
	}

	if len(doc.TimeSeries[0].Period.Points) != 1 {
		t.Errorf("Expected 1 Point, got %d", len(doc.TimeSeries[0].Period.Points))
	}

	if doc.TimeSeries[0].Period.Points[0].PriceAmount != 50.00 {
		t.Errorf("Expected price 50.00, got %f", doc.TimeSeries[0].Period.Points[0].PriceAmount)
	}
}

func TestAcknowledgementMarketDocumentUnmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Acknowledgement_MarketDocument xmlns="urn:iec62325.351:tc57wg16:451-6:acknowledgementdocument:7:3">
	<mRID>error-id</mRID>
	<createdDateTime>2023-01-01T00:00:00Z</createdDateTime>
	<sender_MarketParticipant.mRID codingScheme="A01">10X1001A1001A450</sender_MarketParticipant.mRID>
	<sender_MarketParticipant.marketRole.type>A32</sender_MarketParticipant.marketRole.type>
	<receiver_MarketParticipant.mRID codingScheme="A01">10X1001A1001A450</receiver_MarketParticipant.mRID>
	<receiver_MarketParticipant.marketRole.type>A33</receiver_MarketParticipant.marketRole.type>
	<Reason>
		<code>999</code>
		<text>No matching data found</text>
	</Reason>
</Acknowledgement_MarketDocument>`

	var doc AcknowledgementMarketDocument
	err := xml.Unmarshal([]byte(xmlData), &doc)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if doc.MRID != "error-id" {
		t.Errorf("Expected mRID 'error-id', got '%s'", doc.MRID)
	}

	if doc.Reason.Code != "999" {
		t.Errorf("Expected reason code '999', got '%s'", doc.Reason.Code)
	}

	if doc.Reason.Text != "No matching data found" {
		t.Errorf("Expected reason text 'No matching data found', got '%s'", doc.Reason.Text)
	}
}