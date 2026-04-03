// Copyright 2022-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func benchmarkBuildModelHotdogNode(b *testing.B) *yaml.Node {
	b.Helper()

	yml := `name: yummy
valueName: yammy
beef: true
fat: 200
ketchup: 200.45
mustard: 324938249028.98234892374892374923874823974
grilled: true
maxTemp: 250
maxTempAlt: [1,2,3,4,5]
maxTempHigh: 7392837462032342
drinks:
  - nice
  - rice
  - spice
sides:
  - 0.23
  - 22.23
  - 99.45
  - 22311.2234
bigSides:
  - 98237498.9872349872349872349872347982734927342983479234234234234234234
  - 9827347234234.982374982734987234987
  - 234234234.234982374982347982374982374982347
  - 987234987234987234982734.987234987234987234987234987234987234987234982734982734982734987234987234987234987
temps:
  - 1
  - 2
highTemps:
  - 827349283744710
  - 11732849090192923
buns:
 - true
 - false
unknownElements:
  well:
    whoKnows: not me?
  doYou:
    love: beerToo?
lotsOfUnknowns:
  - wow:
      what: aTrip
  - amazing:
      french: fries
  - amazing:
      french: fries
where:
  things:
    are:
      wild: out here
  howMany:
    bears: 200
there:
  oh: yeah
  care: bear
allTheThings:
  beer: isGood
  cake: isNice`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yml), &root); err != nil {
		b.Fatalf("failed to unmarshal benchmark model: %v", err)
	}
	if len(root.Content) == 0 || root.Content[0] == nil {
		b.Fatal("failed to unmarshal benchmark model: empty root")
	}
	return root.Content[0]
}

func BenchmarkBuildModel_Hotdog(b *testing.B) {
	rootNode := benchmarkBuildModelHotdogNode(b)

	var hd hotdog
	if err := BuildModel(rootNode, &hd); err != nil {
		b.Fatalf("benchmark setup failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hd = hotdog{}
		if err := BuildModel(rootNode, &hd); err != nil {
			b.Fatalf("build model failed: %v", err)
		}
	}
}
