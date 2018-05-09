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

import "github.com/antchfx/xpath"
import "fmt"

func l2x(u uint) xpath.NodeType {
	switch u{
	case RootNode: return xpath.RootNode
	case ElementNode: return xpath.ElementNode
	case TextNode: return xpath.TextNode
	case CommentNode: return xpath.CommentNode
	}
	return xpath.CommentNode /* Bail! */
}

func x2l(u xpath.NodeType) uint {
	switch u{
	case xpath.RootNode: return RootNode
	case xpath.ElementNode: return ElementNode
	case xpath.TextNode: return TextNode
	case xpath.CommentNode: return CommentNode
	}
	return CommentNode /* Bail! */
}

func (n *Node) AppendRoot(src xpath.NodeNavigator) (err error) {
	src.MoveToRoot()
	return n.AppendCurrent(src)
}
func (n *Node) AppendCurrent(src xpath.NodeNavigator) (err error) {
	defer func(){
		rec := recover()
		if rec!=nil {
			if e,ok := rec.(error); ok { err = e } else { err = fmt.Errorf("%v",rec) }
		}
	}()
	
	n.appendCurrent(src)
	return
}
func (n *Node) appendChilds(src xpath.NodeNavigator) {
	if src.MoveToChild() {
		defer src.MoveToParent()
		n.appendCurrent(src)
		for src.MoveToNext() {
			n.appendCurrent(src)
		}
	}
}
func (n *Node) appendCurrent(src xpath.NodeNavigator) {
	nt := x2l(src.NodeType())
	if nt==RootNode { n.appendChilds(src); return }
	info := &NodeInfo{Type:nt}
	info.LocalName = src.LocalName()
	info.Prefix    = src.Prefix()
	sn,err := n.AppendNode(info)
	if err!=nil { panic(err) }
	if nt==ElementNode {
		attr := false
		for src.MoveToNextAttribute() {
			attr = true
			err = sn.AppendAttrib(&Attr{Name: src.LocalName(),Value: src.Value()})
			if err!=nil { break }
		}
		if attr { src.MoveToParent() }
		if err!=nil { panic(err) }
		if src.MoveToChild() {
			defer src.MoveToParent()
			sn.appendCurrent(src)
			for src.MoveToNext() {
				sn.appendCurrent(src)
			}
		}
	} else {
		sn.SetValue(src.Value())
	}
	
}

