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
import "bytes"

type NodeNavigator struct{
	Root, Current *Node
	AttrD int64
	AttrV *Attr
}

var _ xpath.NodeNavigator = new(NodeNavigator)

func CreateXPathNavigator(root *Node) *NodeNavigator {
	return &NodeNavigator{root,root,-1,nil}
}


func (n *NodeNavigator) NodeType() xpath.NodeType {
	if n.AttrD>=0 { return xpath.AttributeNode }
	return l2x(n.Current.Type)
}

func (n *NodeNavigator) LocalName() string{
	if n.AttrV!=nil { return n.AttrV.Name }
	return n.Current.LocalName
}

func (n *NodeNavigator) Prefix() string { return n.Current.Prefix }

func (n *NodeNavigator) Value() string {
	if n.AttrV!=nil {
		return n.AttrV.Value
	}
	if n.Current.Type==TextNode {
		return n.Current.Value() /* Fast path. */
	}
	b := new(bytes.Buffer)
	n.Current.TraverseText(b)
	return b.String()
}

func (n *NodeNavigator) Copy() xpath.NodeNavigator {
	nn := new(NodeNavigator)
	*nn = *n
	return nn
}

func (n *NodeNavigator) MoveToRoot() {
	n.Current = n.Root
}
func (n *NodeNavigator) MoveToParent() bool {
	if nc := n.Current.Parent; nc!=nil { n.Current = nc; return true }
	return false
}
func (n *NodeNavigator) MoveToNextAttribute() bool {
	if n.Current.Type!=ElementNode { return false }
	var ok bool
	n.AttrV,n.AttrD,ok = n.Current.GetAttrib(n.AttrD+1)
	return ok
}
func (n *NodeNavigator) MoveToChild() bool {
	nn,_ := n.Current.Child()
	if nn==nil { return false }
	n.Current = nn
	return true
}
func (n *NodeNavigator) move(whence uint) bool {
	nn,_ := n.Current.move(whence)
	if nn==nil { return false }
	n.Current = nn
	return true
}
func (n *NodeNavigator) MoveToNext() bool { return n.move(0) }
func (n *NodeNavigator) MoveToPrevious() bool { return n.move(1) }
func (n *NodeNavigator) MoveToFirst() bool { return n.move(2) }
func (n *NodeNavigator) MoveTo(o xpath.NodeNavigator) bool {
	*n = *(o.(*NodeNavigator))
	return true
}


