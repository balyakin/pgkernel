package checks

import "github.com/ebalyakin/pgkernel/internal/checker"

func All() []checker.Check {
	all := make([]checker.Check, 0, 13)
	all = append(all, KernelChecks()...)
	all = append(all, MemoryHugePageChecks()...)
	all = append(all, MemorySwapChecks()...)
	all = append(all, IOChecks()...)
	all = append(all, NetworkChecks()...)
	all = append(all, PostgresChecks()...)
	return all
}
