package list

type list interface {
	Destory(interface{})
	Equal(interface{}, interface{}) bool
}

type List struct {
	Head *Element
	Tail *Element
	Size int
	list
}

type Element struct {
	Data interface{}
	Pre  *Element
	Next *Element
}

func NewList(l list) *List {
	return &List{
		Head: nil,
		Tail: nil,
		Size: 0,
		list: l,
	}
}

func (this *List) Insert(pree *Element, data interface{}) (e *Element) {
	e = &Element{Data: data}

	if pree == nil {
		e.Next = this.Head
		if this.Head != nil {
			this.Head.Pre = e
		}
		this.Head = e
		if this.Tail == nil {
			this.Tail = this.Head
		}
	} else {
		e.Next = pree.Next
		if pree.Next != nil {
			pree.Next.Pre = e
		} else {
			this.Tail = e
		}
		pree.Next = e
		e.Pre = pree
	}
	this.Size++
	return
}

func (this *List) Delete(ele *Element) {
	if ele == nil || this.Size == 0 {
		return
	}

	if ele.Next == nil && ele.Pre == nil {
		this.Head = nil
		this.Tail = nil
	} else if ele.Next == nil {
		this.Tail = ele.Pre
		if ele.Pre != nil {
			ele.Pre.Next = nil
		}
	} else if ele.Pre == nil {
		this.Head = ele.Next
		if ele.Next != nil {
			ele.Next.Pre = nil
		}
	} else {
		ele.Next.Pre = ele.Pre
		ele.Pre.Next = ele.Next
	}
	this.Destory(ele.Data)
	this.Size--
	return
}

func (this *List) Update(old interface{}, new interface{}) {
	ele := this.Find(old)
	if ele != nil {
		ele.Data = new
	}
}

func (this *List) Find(d interface{}) (e *Element) {
	for e = this.Head; e != nil; e = e.Next {
		if this.Equal(e.Data, d) {
			break
		}
	}
	return
}
