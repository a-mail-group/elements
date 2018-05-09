/*
This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <http://unlicense.org>
*/
package elements


import (
	"encoding/binary"
	"github.com/coreos/bbolt"
	"github.com/vmihailenco/msgpack"
	"errors"
	"fmt"
	"io"
	"encoding/xml"
)

const (
	// RootNode is a root node of the XML document or node tree.
	RootNode uint = iota

	// ElementNode is an element, such as <element>.
	ElementNode

	// TextNode is the text content of a node.
	TextNode

	// CommentNode is a comment node, such as <!-- my comment -->
	CommentNode
)

/*

A root bucket.

bucket['.info'] -> (binary)Static info
bucket['/'+num] -> (tree)child element
bucket['@'+num] -> (binary)attrv
*/

var (
	EBadNode = errors.New("EBadNode")
	EInvalid = errors.New("EInvalid")
	EOverflow = errors.New("EOverflow")
)

func dbgout(i ...interface{}){
	fmt.Println(i...)
}

func isOf(str []byte,b byte) bool {
	if len(str)==0 { return false }
	return str[0]==b
}

type Attr struct {
	_msgpack struct{} `msgpack:",asArray"`
	Name  string
	Value string
}

type NodeInfo struct {
	_msgpack struct{} `msgpack:",asArray"`
	Type       uint
	LocalName  string
	Prefix     string
}

type Node struct {
	NodeInfo
	Store    *bolt.Bucket
	Parent   *Node
	Number   uint64
}

var (
	b_info = []byte(".info")
	b_value = []byte(".value")
)

func CreateNode(tx *bolt.Tx,name []byte) (node *Node,err error) {
	var bkt *bolt.Bucket
	var meta []byte
	node = new(Node)
	node.Type = RootNode
	meta,err = msgpack.Marshal(&(node.NodeInfo))
	if err!=nil { return }
	bkt,err = tx.CreateBucket(name)
	if err!=nil { return }
	err = bkt.Put(b_info,meta)
	node.Store = bkt
	return
}

func ToNode(b *bolt.Bucket, parent *Node, num uint64) (node *Node,err error){
	node = new(Node)
	err = msgpack.Unmarshal(b.Get(b_info),&(node.NodeInfo))
	node.Store = b
	node.Parent = parent
	node.Number = num
	return
}

func (node *Node) move(whence uint) (*Node,error) {
	if node.Type==RootNode { return nil,nil } /* No parent. No sibling. */
	b := make([]byte,9)
	b[0] = '/'
	n := node.Number
	switch whence {
	case 0: n++
	case 1: n--
	case 2: n = 0
	}
	binary.BigEndian.PutUint64(b[1:],n)
	k,_ := node.Parent.Store.Cursor().Seek(b)
	if !isOf(k,'/') { return nil,nil }
	
	nxt := node.Parent.Store.Bucket(k)
	return ToNode(nxt,node.Parent,binary.BigEndian.Uint64(k[1:]))
}
func (node *Node) Next() (*Node,error) { return node.move(0) }
func (node *Node) Prev() (*Node,error) { return node.move(1) }
func (node *Node) First() (*Node,error) { return node.move(2) }
func (node *Node) Child() (*Node,error) {
	switch node.Type {
	case ElementNode,RootNode: break
	default: return nil,nil /* No childs. */
	}
	b := make([]byte,9)
	b[0] = '/'
	binary.BigEndian.PutUint64(b[1:],0)
	k,_ := node.Store.Cursor().Seek(b)
	if !isOf(k,'/') { return nil,nil }
	
	nxt := node.Store.Bucket(k)
	return ToNode(nxt,node,binary.BigEndian.Uint64(k[1:]))
}

func (node *Node) AppendAttrib(attr *Attr) error {
	data,err := msgpack.Marshal(attr)
	if err!=nil { return err }
	if node.Type!=ElementNode { return EOverflow } /* No attributes. */
	b := make([]byte,9)
	b[0] = '@'
	
	binary.BigEndian.PutUint64(b[1:],0x7FFFFFFFFFFFFFFF)
	c := node.Store.Cursor()
	
	k,_ := c.Seek(b)
	if len(k)>0 { k,_ = c.Prev() } else { k,_ = c.Last() }
	
	num := int64(0)
	
	if isOf(k,'@') {
		num = int64(binary.BigEndian.Uint64(k[1:]))
		num++
		if num<0 { return EOverflow }
	}
	
	binary.BigEndian.PutUint64(b[1:],uint64(num))
	
	return node.Store.Put(b,data)
}
func (node *Node) GetAttrib(attr int64) (*Attr,int64,bool) {
	if node.Type!=ElementNode { return nil,-1,false } /* No attributes. */
	b := make([]byte,9)
	b[0] = '@'
	
	binary.BigEndian.PutUint64(b[1:],uint64(attr))
	k,v := node.Store.Cursor().Seek(b)
	if !isOf(k,'@') { return nil,-1,false }
	
	kv := new(Attr)
	err := msgpack.Unmarshal(v,kv)
	if err!=nil { panic(err) }
	nattr := int64(binary.BigEndian.Uint64(b[1:]))
	
	return kv,nattr,true
}
func (node *Node) Value() string {
	return string(node.Store.Get(b_value))
}
func (node *Node) SetValue(s string) error {
	return node.Store.Put(b_value,[]byte(s))
}
func (node *Node) AppendNode(n *NodeInfo) (*Node,error) {
	if n.Type==RootNode { return nil,EInvalid }
	switch node.Type {
	case ElementNode,RootNode: break
	default: return nil,EBadNode /* No childs. */
	}
	b := make([]byte,9)
	b[0] = '/'
	
	meta,err := msgpack.Marshal(n)
	if err!=nil { return nil,err }
	
	num,err := node.Store.NextSequence()
	if err!=nil { return nil,err }
	
	binary.BigEndian.PutUint64(b[1:],num)
	
	bkt,err := node.Store.CreateBucket(b)
	if err!=nil { return nil,err }
	
	err = bkt.Put(b_info,meta)
	if err!=nil { return nil,err }
	
	return ToNode(bkt,node,num)
}
func (node *Node) TraverseText(w io.Writer) {
	switch node.Type {
	case TextNode:
		w.Write(node.Store.Get(b_value))
	case ElementNode,RootNode:
		cur := node.Store.Cursor()
		b := make([]byte,9)
		b[0] = '/'
		binary.BigEndian.PutUint64(b[1:],0)
		for k,_ := cur.Seek(b); isOf(k,'/'); k,_ = cur.Next() {
			n,e := ToNode(node.Store.Bucket(k),node,binary.BigEndian.Uint64(k[1:]))
			if e!=nil { continue }
			n.TraverseText(w)
		}
	}
}
func (node *Node) TraverseXml(w io.Writer){
	switch node.Type {
	case TextNode:
		xml.EscapeText(w,node.Store.Get(b_value))
	case CommentNode:
		fmt.Fprint(w,"<!--")
		xml.EscapeText(w,node.Store.Get(b_value))
		fmt.Fprint(w,"-->")
	case RootNode, ElementNode:
		xename := node.LocalName
		b := make([]byte,9)
		
		if node.Type==ElementNode {
			if node.Prefix!="" {
				xename = node.Prefix+":"+xename
			}
			fmt.Fprintf(w,"<%s",xename)
			b[0] = '@'
			
			binary.BigEndian.PutUint64(b[1:],0)
			cur := node.Store.Cursor()
			kv := new(Attr)
			for k,v := cur.Seek(b); isOf(k,'@'); k,v = cur.Next() {
				e := msgpack.Unmarshal(v,kv)
				if e!=nil { continue }
				fmt.Fprintf(w," %s=\"",kv.Name)
				xml.EscapeText(w,[]byte(kv.Value))
				fmt.Fprint(w,"\"")
			}
			fmt.Fprint(w,">")
		}
		
		cur := node.Store.Cursor()
		b[0] = '/'
		// binary.BigEndian.PutUint64(b[1:],0)
		for k,_ := cur.Seek(b); isOf(k,'/'); k,_ = cur.Next() {
			n,e := ToNode(node.Store.Bucket(k),node,binary.BigEndian.Uint64(k[1:]))
			if e!=nil { continue }
			n.TraverseXml(w)
		}
		
		if node.Type==ElementNode {
			fmt.Fprintf(w,"</%s>",xename)
		}
	}
}


//func (node *Node) 


