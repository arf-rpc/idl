package ast

type FileSet []File

type File struct {
	Path string
	Tree *Tree
}
