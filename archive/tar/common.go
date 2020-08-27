// 版权所有2009 The Go Authors。 版权所有。
// 此源代码的使用受BSD样式的约束
// 可以在LICENSE文件中找到的许可证。

// Package tar 实现对tar存档的访问。
//
// 磁带存档（tar）是一种文件格式，用于存储可以以流方式读写的文件序列。
// 此软件包旨在涵盖该格式的大多数变体，包括GNU和BSD tar工具产生的那些变体。
package tar

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// 错误：在Header中使用Uid和Gid字段可能会在32位体系结构上溢出。 如果解码时遇到较大的值，则存储在Header中的结果将为截断的版本。
var (
	ErrHeader          = errors.New("archive/tar: invalid tar header")                       // 错误："无效的tar header"
	ErrWriteTooLong    = errors.New("archive/tar: write too long")                           // 错误："写入太长"
	ErrFieldTooLong    = errors.New("archive/tar: header field too long")                    // 错误："header太长"
	ErrWriteAfterClose = errors.New("archive/tar: write after close")                        // 错误："在关闭之后写入"
	errMissData        = errors.New("archive/tar: sparse file references non-existent data") // 错误："稀疏文件引用了不存在的数据"
	errUnrefData       = errors.New("archive/tar: sparse file contains unreferenced data")   // 错误："稀疏文件包含未引用的数据"
	errWriteHole       = errors.New("archive/tar: write non-NUL byte in sparse hole")        // 错误："在稀疏孔中写入非NUL字节"
)

type headerError []string

func (he headerError) Error() string { // 注：输出错误he
	const prefix = "archive/tar: cannot encode header" // 注：无法编码标头
	var ss []string
	for _, s := range he { // 注：遍历he，将he转为[]string
		if s != "" {
			ss = append(ss, s)
		}
	}
	if len(ss) == 0 {
		return prefix
	}
	return fmt.Sprintf("%s: %v", prefix, strings.Join(ss, "; and ")) // 注：输出"archive/tar: cannot encode header: error1; and error2"
}

// Header.Typeflag的类型标志。
const (
	// 类型'0'表示常规文件。
	TypeReg  = '0'
	TypeRegA = '\x00' // 不推荐使用：请改用TypeReg。

	// 类型'1'到'6'是仅标头标志，并且可能没有数据主体。
	TypeLink    = '1' // 硬链接
	TypeSymlink = '2' // 符号链接
	TypeChar    = '3' // 角色设备节点
	TypeBlock   = '4' // 块设备节点
	TypeDir     = '5' // 目录
	TypeFifo    = '6' // FIFO节点

	// 类型'7'被保留。
	TypeCont = '7'

	// PAX格式使用类型'x'来存储仅与下一个文件相关的键值记录。
	// 此包透明地处理这些类型。
	TypeXHeader = 'x'

	// PAX格式使用类型'g'来存储与所有后续文件相关的键值记录。
	// 此包仅支持解析和组合此类标头，但当前不支持跨文件持久化全局状态。
	TypeXGlobalHeader = 'g'

	// 类型'S'表示GNU格式的稀疏文件。
	TypeGNUSparse = 'S'

	// GNU格式将'L'和'K'类型用于元文件，该元文件用于存储下一个文件的路径或链接名称。
	// 此包透明地处理这些类型。
	TypeGNULongName = 'L'
	TypeGNULongLink = 'K'
)

// PAX扩展头记录的关键字。
const (
	paxNone     = "" // 表示没有合适的PAX键
	paxPath     = "path"
	paxLinkpath = "linkpath"
	paxSize     = "size"
	paxUid      = "uid"
	paxGid      = "gid"
	paxUname    = "uname"
	paxGname    = "gname"
	paxMtime    = "mtime"
	paxAtime    = "atime"
	paxCtime    = "ctime"   // 已从PAX规范的更高版本中删除，但有效
	paxCharset  = "charset" // 目前未使用
	paxComment  = "comment" // 目前未使用

	paxSchilyXattr = "SCHILY.xattr."

	// PAX扩展头中GNU稀疏文件的关键字。
	paxGNUSparse          = "GNU.sparse."
	paxGNUSparseNumBlocks = "GNU.sparse.numblocks"
	paxGNUSparseOffset    = "GNU.sparse.offset"
	paxGNUSparseNumBytes  = "GNU.sparse.numbytes"
	paxGNUSparseMap       = "GNU.sparse.map"
	paxGNUSparseName      = "GNU.sparse.name"
	paxGNUSparseMajor     = "GNU.sparse.major"
	paxGNUSparseMinor     = "GNU.sparse.minor"
	paxGNUSparseSize      = "GNU.sparse.size"
	paxGNUSparseRealSize  = "GNU.sparse.realsize"
)

// basicKeys 是我们内置支持的一组PAX密钥。
// 它不包含"字符集"或"注释"，它们都是PAX特定的，因此不太可能将它们添加为Header的一流功能。
// 用户可以使用PAXRecords字段自行设置。
var basicKeys = map[string]bool{
	paxPath: true, paxLinkpath: true, paxSize: true, paxUid: true, paxGid: true,
	paxUname: true, paxGname: true, paxMtime: true, paxAtime: true, paxCtime: true,
}

// Header 代表tar归档文件中的单个标头。
// 某些字段可能未填充。
//
// 为了向前兼容，用户从Reader中检索Header，然后以某种方式对其进行变异，
// 然后将其传递回Writer.WriteHeader应该通过创建新Header并复制他们想要保留的字段来实现。
type Header struct {
	// Typeflag是标题条目的类型。
	// 零值会根据Name中是否存在斜杠而自动提升为TypeReg或TypeDir。
	Typeflag byte

	Name     string // 文件入口名称
	Linkname string // 链接的目标名称（对TypeLink或TypeSymlink有效）

	Size  int64  // 逻辑文件大小（以字节为单位）
	Mode  int64  // 权限和模式位
	Uid   int    // 所有者的用户标识
	Gid   int    // 所有者的组ID
	Uname string // 所有者的用户名
	Gname string // 所有者的组名

	// 如果未指定Format，则Writer.WriteHeader将ModTime舍入到最接近的秒数，并忽略AccessTime和ChangeTime字段。
	//
	// 要使用AccessTime或ChangeTime，请将格式指定为PAX或GNU。
	// 要使用亚秒级分辨率，请将格式指定为PAX。
	ModTime    time.Time // 修改时间
	AccessTime time.Time // 访问时间（需要PAX或GNU支持）
	ChangeTime time.Time // 更改时间（需要PAX或GNU支持）

	Devmajor int64 // 主设备号（对TypeChar或TypeBlock有效）
	Devminor int64 // 次设备号（对TypeChar或TypeBlock有效）

	// Xattrs 将扩展属性作为PAX记录存储在命名空间"SCHILY.xattr"下。
	//
	// 以下在语义上是等效的：
	//  h.Xattrs[key] = value
	//  h.PAXRecords["SCHILY.xattr."+key] = value
	//
	// 调用Writer.WriteHeader时，Xattrs的内容将优先于PAXRecords中的内容。
	//
	// 不推荐使用：请改用PAXRecords。
	Xattrs map[string]string

	// PAXRecords 是PAX扩展头记录的映射。
	//
	// 用户定义的记录应具有以下格式的键：
	// 	VENDOR.keyword
	// 其中VENDOR是所有大写形式的名称空间，而关键字可能不包含'='字符（例如，"GOLANG.pkg.version")
	// 键和值应为非空的UTF-8字符串。
	//
	// 调用Writer.WriteHeader时，从Header中其他字段派生的PAX记录优先于PAXRecords。
	PAXRecords map[string]string

	// Format 指定tar标头的格式。
	//
	// 这是由Reader.Next设置的，是对格式的最大努力。
	// 由于Reader可以自由读取一些不兼容的文件，因此有可能是FormatUnknown。
	//
	// 如果在调用Writer.WriteHeader时未指定格式，则它将使用能够对该Header进行编码的第一种格式（按USTAR，PAX，GNU的顺序）（请参见Format）。
	Format Format
}

// sparseEntry 表示文件中Offset处的Length长度的片段。
type sparseEntry struct{ Offset, Length int64 }

func (s sparseEntry) endOffset() int64 { return s.Offset + s.Length } // 注：获取s偏移结束的位置

// 稀疏文件可以表示为sparseDatas或sparseHoles。
// 只要知道总大小，它们就相等，并且可以将其转换为另一种形式并返回。
// 支持稀疏文件的各种tar格式以sparseDatas形式表示稀疏文件。
// 也就是说，它们指定文件中具有数据的片段，并将其他所有内容都视为具有零字节。
// 这样，此程序包中的编码和解码逻辑处理sparseDatas。
//
// 但是，外部API使用sparseHoles而不是sparseDatas，
// 因为sparseHoles的零值在逻辑上表示一个普通文件（即其中没有孔）。
// 另一方面，sparseDatas的零值表示文件中没有数据，这很奇怪。
//
// 作为示例，如果基础原始文件包含10字节数据：
//	var compactFile = "abcdefgh"
//
// 并且稀疏映射具有以下条目：
//	var spd sparseDatas = []sparseEntry{
//		{Offset: 2,  Length: 5},  // 2..6的数据片段
//		{Offset: 18, Length: 3},  // 18..20的数据片段
//	}
//	var sph sparseHoles = []sparseEntry{
//		{Offset: 0,  Length: 2},  // 0..1的孔片段
//		{Offset: 7,  Length: 11}, // 7..17的孔片段
//		{Offset: 21, Length: 4},  // 21..24的孔片段
//	}
//
// 然后，结果Header.Size为25的稀疏文件的内容为：
//	var sparseFile = "\x00"*2 + "abcde" + "\x00"*11 + "fgh" + "\x00"*4
//
// 例：var a = make([]int, 30)
//     a[2] = 1
//     a[3] = 1
//     a[4] = 1
//     a[5] = 1
//     a[6] = 1
//     a[18] = 1
//     a[19] = 1
//     a[20] = 1
// a[0]、a[1]为孔，数据为{Offset: 0,  Length: 2}，表示第0个元素开始的2个元素为孔
// a[18]、a[19]、a[20]为数据，数据为{Offset: 18, Length: 3}，表示第18个元素开始的3个元素为数据
type (
	sparseDatas []sparseEntry
	sparseHoles []sparseEntry
)

// validateSparseEntries 报告sp是否为有效的稀疏映射。
// sp表示数据片段还是孔片段都没有关系。
func validateSparseEntries(sp []sparseEntry, size int64) bool { // 注：获取sp是否为有效的偏移不超过size的稀疏映射
	// 验证所有稀疏条目。 这些检查与BSD tar实用程序执行的检查相同。
	if size < 0 {
		return false
	}
	var pre sparseEntry
	for _, cur := range sp { // 注：遍历sp
		switch {
		case cur.Offset < 0 || cur.Length < 0: // 注：偏移或长度 < 0
			return false // 负值永远为false
		case cur.Offset > math.MaxInt64-cur.Length: // 注：偏移结尾溢出
			return false // 大整数溢出
		case cur.endOffset() > size: // 注：偏移结尾超过了size
			return false // 区域超出实际大小
		case pre.endOffset() > cur.Offset: // 注：上一个条目的偏移结尾 > 此条目的偏移
			return false // 区域不能重叠且必须有序
		}
		pre = cur // 注：sp中的条目中的偏移必须比前一个条目大
	}
	return true
}

// alignSparseEntries 使src突变并返回dst，其中每个片段的起始偏移量一直对齐到最近的块边缘，而每个结束偏移量都向下对齐到最近的块边缘。
//
// 即使Go tar阅读器和BSD tar实用程序可以处理具有任意偏移量和长度的条目，GNU tar实用程序也只能处理为blockSize倍数的偏移量和长度。
func alignSparseEntries(src []sparseEntry, size int64) []sparseEntry { // 注：#
	dst := src[:0]
	for _, s := range src { // 注：遍历src
		pos, end := s.Offset, s.endOffset()
		pos += blockPadding(+pos) // 向上舍入到最接近的块大小，注：#
		if end != size {
			end -= blockPadding(-end) // 四舍五入到最接近的块大小
		}
		if pos < end {
			dst = append(dst, sparseEntry{Offset: pos, Length: end - pos})
		}
	}
	return dst
}

// invertSparseEntries 将稀疏映射从一种形式转换为另一种形式。
// 如果输入为sparseHoles，则它将输出sparseDatas，反之亦然。
// 输入必须已经验证。
//
// 此函数会更改src并返回规范化的映射，其中：
// 	*相邻的片段合并在一起
// 	*只有最后一个片段可能为空
//	 *最后一个片段的endOffset是总大小
func invertSparseEntries(src []sparseEntry, size int64) []sparseEntry { // 注：将src在sparseHoles与sparseDatas之间互相转换
	dst := src[:0]
	var pre sparseEntry
	for _, cur := range src { // 注：遍历src
		if cur.Length == 0 {
			continue // 跳过空片段
		}
		pre.Length = cur.Offset - pre.Offset // 注：此条目的偏移 - 上一个条目的偏移结尾
		if pre.Length > 0 {
			dst = append(dst, pre) // 仅添加非空片段
		}
		pre.Offset = cur.endOffset()
	}
	pre.Length = size - pre.Offset // 可能是唯一的空片段
	return append(dst, pre)
}

// fileState 跟踪当前文件剩余的逻辑（包括稀疏漏洞）和物理（实际在tar存档中）字节的数量。
//
// 不变式：LogicalRemaining> = PhysicalRemaining
type fileState interface {
	LogicalRemaining() int64
	PhysicalRemaining() int64
}

// allowedFormats 确定可以使用的格式。
// 返回的值是多种可能格式的逻辑或。
// 如果值为FormatUnknown，则无法对输入Header进行编码，并返回错误说明原因。
//
//作为检查字段的副产品，此函数返回paxHdrs，其中包含所有无法直接编码的字段。
// 值接收器确保此方法不会使源Header发生变异。
func (h Header) allowedFormats() (format Format, paxHdrs map[string]string, err error) { // 注：#
	format = FormatUSTAR | FormatPAX | FormatGNU
	paxHdrs = make(map[string]string)

	var whyNoUSTAR, whyNoPAX, whyNoGNU string
	var preferPAX bool // 在USTAR上优先使用PAX
	verifyString := func(s string, size int, name, paxKey string) {
		// NUL终止符对于路径和链接路径是可选的。
		// 从技术上讲，uname和gname是必需的，但是GNU和BSD tar都不会对其进行检查。
		tooLong := len(s) > size
		allowLongGNU := paxKey == paxPath || paxKey == paxLinkpath
		if hasNUL(s) || (tooLong && !allowLongGNU) {
			whyNoGNU = fmt.Sprintf("GNU cannot encode %s=%q", name, s)
			format.mustNotBe(FormatGNU)
		}
		if !isASCII(s) || tooLong {
			canSplitUSTAR := paxKey == paxPath
			if _, _, ok := splitUSTARPath(s); !canSplitUSTAR || !ok {
				whyNoUSTAR = fmt.Sprintf("USTAR cannot encode %s=%q", name, s)
				format.mustNotBe(FormatUSTAR)
			}
			if paxKey == paxNone {
				whyNoPAX = fmt.Sprintf("PAX cannot encode %s=%q", name, s)
				format.mustNotBe(FormatPAX)
			} else {
				paxHdrs[paxKey] = s
			}
		}
		if v, ok := h.PAXRecords[paxKey]; ok && v == s {
			paxHdrs[paxKey] = v
		}
	}
	verifyNumeric := func(n int64, size int, name, paxKey string) {
		if !fitsInBase256(size, n) {
			whyNoGNU = fmt.Sprintf("GNU cannot encode %s=%d", name, n)
			format.mustNotBe(FormatGNU)
		}
		if !fitsInOctal(size, n) {
			whyNoUSTAR = fmt.Sprintf("USTAR cannot encode %s=%d", name, n)
			format.mustNotBe(FormatUSTAR)
			if paxKey == paxNone {
				whyNoPAX = fmt.Sprintf("PAX cannot encode %s=%d", name, n)
				format.mustNotBe(FormatPAX)
			} else {
				paxHdrs[paxKey] = strconv.FormatInt(n, 10)
			}
		}
		if v, ok := h.PAXRecords[paxKey]; ok && v == strconv.FormatInt(n, 10) {
			paxHdrs[paxKey] = v
		}
	}
	verifyTime := func(ts time.Time, size int, name, paxKey string) {
		if ts.IsZero() {
			return // Always okay
		}
		if !fitsInBase256(size, ts.Unix()) {
			whyNoGNU = fmt.Sprintf("GNU cannot encode %s=%v", name, ts)
			format.mustNotBe(FormatGNU)
		}
		isMtime := paxKey == paxMtime
		fitsOctal := fitsInOctal(size, ts.Unix())
		if (isMtime && !fitsOctal) || !isMtime {
			whyNoUSTAR = fmt.Sprintf("USTAR cannot encode %s=%v", name, ts)
			format.mustNotBe(FormatUSTAR)
		}
		needsNano := ts.Nanosecond() != 0
		if !isMtime || !fitsOctal || needsNano {
			preferPAX = true // USTAR may truncate sub-second measurements
			if paxKey == paxNone {
				whyNoPAX = fmt.Sprintf("PAX cannot encode %s=%v", name, ts)
				format.mustNotBe(FormatPAX)
			} else {
				paxHdrs[paxKey] = formatPAXTime(ts)
			}
		}
		if v, ok := h.PAXRecords[paxKey]; ok && v == formatPAXTime(ts) {
			paxHdrs[paxKey] = v
		}
	}

	// Check basic fields.
	var blk block
	v7 := blk.V7()
	ustar := blk.USTAR()
	gnu := blk.GNU()
	verifyString(h.Name, len(v7.Name()), "Name", paxPath)
	verifyString(h.Linkname, len(v7.LinkName()), "Linkname", paxLinkpath)
	verifyString(h.Uname, len(ustar.UserName()), "Uname", paxUname)
	verifyString(h.Gname, len(ustar.GroupName()), "Gname", paxGname)
	verifyNumeric(h.Mode, len(v7.Mode()), "Mode", paxNone)
	verifyNumeric(int64(h.Uid), len(v7.UID()), "Uid", paxUid)
	verifyNumeric(int64(h.Gid), len(v7.GID()), "Gid", paxGid)
	verifyNumeric(h.Size, len(v7.Size()), "Size", paxSize)
	verifyNumeric(h.Devmajor, len(ustar.DevMajor()), "Devmajor", paxNone)
	verifyNumeric(h.Devminor, len(ustar.DevMinor()), "Devminor", paxNone)
	verifyTime(h.ModTime, len(v7.ModTime()), "ModTime", paxMtime)
	verifyTime(h.AccessTime, len(gnu.AccessTime()), "AccessTime", paxAtime)
	verifyTime(h.ChangeTime, len(gnu.ChangeTime()), "ChangeTime", paxCtime)

	// Check for header-only types.
	var whyOnlyPAX, whyOnlyGNU string
	switch h.Typeflag {
	case TypeReg, TypeChar, TypeBlock, TypeFifo, TypeGNUSparse:
		// Exclude TypeLink and TypeSymlink, since they may reference directories.
		if strings.HasSuffix(h.Name, "/") {
			return FormatUnknown, nil, headerError{"filename may not have trailing slash"}
		}
	case TypeXHeader, TypeGNULongName, TypeGNULongLink:
		return FormatUnknown, nil, headerError{"cannot manually encode TypeXHeader, TypeGNULongName, or TypeGNULongLink headers"}
	case TypeXGlobalHeader:
		h2 := Header{Name: h.Name, Typeflag: h.Typeflag, Xattrs: h.Xattrs, PAXRecords: h.PAXRecords, Format: h.Format}
		if !reflect.DeepEqual(h, h2) {
			return FormatUnknown, nil, headerError{"only PAXRecords should be set for TypeXGlobalHeader"}
		}
		whyOnlyPAX = "only PAX supports TypeXGlobalHeader"
		format.mayOnlyBe(FormatPAX)
	}
	if !isHeaderOnlyType(h.Typeflag) && h.Size < 0 {
		return FormatUnknown, nil, headerError{"negative size on header-only type"}
	}

	// Check PAX records.
	if len(h.Xattrs) > 0 {
		for k, v := range h.Xattrs {
			paxHdrs[paxSchilyXattr+k] = v
		}
		whyOnlyPAX = "only PAX supports Xattrs"
		format.mayOnlyBe(FormatPAX)
	}
	if len(h.PAXRecords) > 0 {
		for k, v := range h.PAXRecords {
			switch _, exists := paxHdrs[k]; {
			case exists:
				continue // Do not overwrite existing records
			case h.Typeflag == TypeXGlobalHeader:
				paxHdrs[k] = v // Copy all records
			case !basicKeys[k] && !strings.HasPrefix(k, paxGNUSparse):
				paxHdrs[k] = v // Ignore local records that may conflict
			}
		}
		whyOnlyPAX = "only PAX supports PAXRecords"
		format.mayOnlyBe(FormatPAX)
	}
	for k, v := range paxHdrs {
		if !validPAXRecord(k, v) {
			return FormatUnknown, nil, headerError{fmt.Sprintf("invalid PAX record: %q", k+" = "+v)}
		}
	}

	// TODO(dsnet): Re-enable this when adding sparse support.
	// See https://golang.org/issue/22735
	/*
		// Check sparse files.
		if len(h.SparseHoles) > 0 || h.Typeflag == TypeGNUSparse {
			if isHeaderOnlyType(h.Typeflag) {
				return FormatUnknown, nil, headerError{"header-only type cannot be sparse"}
			}
			if !validateSparseEntries(h.SparseHoles, h.Size) {
				return FormatUnknown, nil, headerError{"invalid sparse holes"}
			}
			if h.Typeflag == TypeGNUSparse {
				whyOnlyGNU = "only GNU supports TypeGNUSparse"
				format.mayOnlyBe(FormatGNU)
			} else {
				whyNoGNU = "GNU supports sparse files only with TypeGNUSparse"
				format.mustNotBe(FormatGNU)
			}
			whyNoUSTAR = "USTAR does not support sparse files"
			format.mustNotBe(FormatUSTAR)
		}
	*/

	// Check desired format.
	if wantFormat := h.Format; wantFormat != FormatUnknown {
		if wantFormat.has(FormatPAX) && !preferPAX {
			wantFormat.mayBe(FormatUSTAR) // PAX implies USTAR allowed too
		}
		format.mayOnlyBe(wantFormat) // Set union of formats allowed and format wanted
	}
	if format == FormatUnknown {
		switch h.Format {
		case FormatUSTAR:
			err = headerError{"Format specifies USTAR", whyNoUSTAR, whyOnlyPAX, whyOnlyGNU}
		case FormatPAX:
			err = headerError{"Format specifies PAX", whyNoPAX, whyOnlyGNU}
		case FormatGNU:
			err = headerError{"Format specifies GNU", whyNoGNU, whyOnlyPAX}
		default:
			err = headerError{whyNoUSTAR, whyNoPAX, whyNoGNU, whyOnlyPAX, whyOnlyGNU}
		}
	}
	return format, paxHdrs, err
}

// FileInfo 返回Header的os.FileInfo。
func (h *Header) FileInfo() os.FileInfo { // 注：返回h的os.FileInfo
	return headerFileInfo{h}
}

// headerFileInfo 实现os.FileInfo。
type headerFileInfo struct {
	h *Header
}

func (fi headerFileInfo) Size() int64        { return fi.h.Size }         // 注：获取Header的Size
func (fi headerFileInfo) IsDir() bool        { return fi.Mode().IsDir() } // 注：
func (fi headerFileInfo) ModTime() time.Time { return fi.h.ModTime }
func (fi headerFileInfo) Sys() interface{}   { return fi.h }

// Name returns the base name of the file.
func (fi headerFileInfo) Name() string {
	if fi.IsDir() {
		return path.Base(path.Clean(fi.h.Name))
	}
	return path.Base(fi.h.Name)
}

// Mode returns the permission and mode bits for the headerFileInfo.
func (fi headerFileInfo) Mode() (mode os.FileMode) {
	// Set file permission bits.
	mode = os.FileMode(fi.h.Mode).Perm()

	// Set setuid, setgid and sticky bits.
	if fi.h.Mode&c_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if fi.h.Mode&c_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	if fi.h.Mode&c_ISVTX != 0 {
		mode |= os.ModeSticky
	}

	// Set file mode bits; clear perm, setuid, setgid, and sticky bits.
	switch m := os.FileMode(fi.h.Mode) &^ 07777; m {
	case c_ISDIR:
		mode |= os.ModeDir
	case c_ISFIFO:
		mode |= os.ModeNamedPipe
	case c_ISLNK:
		mode |= os.ModeSymlink
	case c_ISBLK:
		mode |= os.ModeDevice
	case c_ISCHR:
		mode |= os.ModeDevice
		mode |= os.ModeCharDevice
	case c_ISSOCK:
		mode |= os.ModeSocket
	}

	switch fi.h.Typeflag {
	case TypeSymlink:
		mode |= os.ModeSymlink
	case TypeChar:
		mode |= os.ModeDevice
		mode |= os.ModeCharDevice
	case TypeBlock:
		mode |= os.ModeDevice
	case TypeDir:
		mode |= os.ModeDir
	case TypeFifo:
		mode |= os.ModeNamedPipe
	}

	return mode
}

// sysStat, if non-nil, populates h from system-dependent fields of fi.
var sysStat func(fi os.FileInfo, h *Header) error

const (
	// Mode constants from the USTAR spec:
	// See http://pubs.opengroup.org/onlinepubs/9699919799/utilities/pax.html#tag_20_92_13_06
	c_ISUID = 04000 // Set uid
	c_ISGID = 02000 // Set gid
	c_ISVTX = 01000 // Save text (sticky bit)

	// Common Unix mode constants; these are not defined in any common tar standard.
	// Header.FileInfo understands these, but FileInfoHeader will never produce these.
	c_ISDIR  = 040000  // Directory
	c_ISFIFO = 010000  // FIFO
	c_ISREG  = 0100000 // Regular file
	c_ISLNK  = 0120000 // Symbolic link
	c_ISBLK  = 060000  // Block special file
	c_ISCHR  = 020000  // Character special file
	c_ISSOCK = 0140000 // Socket
)

// FileInfoHeader creates a partially-populated Header from fi.
// If fi describes a symlink, FileInfoHeader records link as the link target.
// If fi describes a directory, a slash is appended to the name.
//
// Since os.FileInfo's Name method only returns the base name of
// the file it describes, it may be necessary to modify Header.Name
// to provide the full path name of the file.
func FileInfoHeader(fi os.FileInfo, link string) (*Header, error) {
	if fi == nil {
		return nil, errors.New("archive/tar: FileInfo is nil")
	}
	fm := fi.Mode()
	h := &Header{
		Name:    fi.Name(),
		ModTime: fi.ModTime(),
		Mode:    int64(fm.Perm()), // or'd with c_IS* constants later
	}
	switch {
	case fm.IsRegular():
		h.Typeflag = TypeReg
		h.Size = fi.Size()
	case fi.IsDir():
		h.Typeflag = TypeDir
		h.Name += "/"
	case fm&os.ModeSymlink != 0:
		h.Typeflag = TypeSymlink
		h.Linkname = link
	case fm&os.ModeDevice != 0:
		if fm&os.ModeCharDevice != 0 {
			h.Typeflag = TypeChar
		} else {
			h.Typeflag = TypeBlock
		}
	case fm&os.ModeNamedPipe != 0:
		h.Typeflag = TypeFifo
	case fm&os.ModeSocket != 0:
		return nil, fmt.Errorf("archive/tar: sockets not supported")
	default:
		return nil, fmt.Errorf("archive/tar: unknown file mode %v", fm)
	}
	if fm&os.ModeSetuid != 0 {
		h.Mode |= c_ISUID
	}
	if fm&os.ModeSetgid != 0 {
		h.Mode |= c_ISGID
	}
	if fm&os.ModeSticky != 0 {
		h.Mode |= c_ISVTX
	}
	// If possible, populate additional fields from OS-specific
	// FileInfo fields.
	if sys, ok := fi.Sys().(*Header); ok {
		// This FileInfo came from a Header (not the OS). Use the
		// original Header to populate all remaining fields.
		h.Uid = sys.Uid
		h.Gid = sys.Gid
		h.Uname = sys.Uname
		h.Gname = sys.Gname
		h.AccessTime = sys.AccessTime
		h.ChangeTime = sys.ChangeTime
		if sys.Xattrs != nil {
			h.Xattrs = make(map[string]string)
			for k, v := range sys.Xattrs {
				h.Xattrs[k] = v
			}
		}
		if sys.Typeflag == TypeLink {
			// hard link
			h.Typeflag = TypeLink
			h.Size = 0
			h.Linkname = sys.Linkname
		}
		if sys.PAXRecords != nil {
			h.PAXRecords = make(map[string]string)
			for k, v := range sys.PAXRecords {
				h.PAXRecords[k] = v
			}
		}
	}
	if sysStat != nil {
		return h, sysStat(fi, h)
	}
	return h, nil
}

// isHeaderOnlyType checks if the given type flag is of the type that has no
// data section even if a size is specified.
func isHeaderOnlyType(flag byte) bool {
	switch flag {
	case TypeLink, TypeSymlink, TypeChar, TypeBlock, TypeDir, TypeFifo:
		return true
	default:
		return false
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
