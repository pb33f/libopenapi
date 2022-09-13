package main

//func main() {
//
//	testData := `openapi: 3.0.1
//info:
//  title: this is a title
//  description: this is a description
//tags:
//  - name: Tag A
//    description: cake
//    x-hack: true
//  - name: Tag B
//    description: coffee
//    x-code: hack`
//
//	data := []byte(testData)
//	_ = ioutil.WriteFile("sample.yaml", data, 0664)
//
//	info, _ := datamodel.ExtractSpecInfo(data)
//	lowDoc, err := low.CreateDocument(info)
//	if len(err) > 0 {
//		for e := range err {
//			fmt.Printf("%e\n", err[e])
//		}
//		return
//	}
//	highDoc := high.NewDocument(lowDoc)
//
//	highDoc.Info.GoLow().Title.ValueNode.Value = "let's hack this"
//	//highDoc.Tags[0].SetName("We are a new name now")
//	//highDoc.Tags[0].SetDescription("and a new description")
//
//	//newTag := lowDoc.AddTag()
//	//fmt.Println(newTag)
//	modified, _ := yaml.Marshal(info.RootNode)
//	fmt.Println(string(modified))
//
//	os.Remove("sample.yaml")
//
//}
