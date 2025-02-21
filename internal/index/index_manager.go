package index

import (
	"crypto/sha256"
)


type IndexManager struct {
	tree 			*AVLTree
	filter 			*BloomFilterManager
	currentIndex	int64
}

func NewIndexManager() *IndexManager {
	return &IndexManager{
		tree: &AVLTree{nil},
		filter: InitFilter(),
		currentIndex: 0, // Corresponds to genesis block
	}
}

func (im *IndexManager) GetIP(domain string) string {
	// Check if the domain is valid
	if !im.filter.IsValid(domain) {
		return ""
	}
	
	targetNode := im.tree.Search(HashDomain(domain))
	if targetNode == nil {
		return ""
	}
	
	return targetNode.value
}

func (im *IndexManager) GetIndexHash() []byte {
	return ComputeIndexNodeHash(im.tree.root)
}

func (im *IndexManager) Add(domain string, ip string) {
	im.tree.Add(HashDomain(domain), ip)
	im.filter.AddToValidList(domain)
}

func (im *IndexManager) Update(oldDomain string, newDomain string, ip string) {
	im.tree.Update(HashDomain(oldDomain), HashDomain(newDomain), ip)
}

func (im *IndexManager) Remove(domain string) {
	im.tree.Remove(HashDomain(domain))
	im.filter.AddToRevocationList(domain)
}

func HashDomain(domain string) string {
	hash := sha256.Sum256([]byte(domain))
	return string(hash[:])
}
