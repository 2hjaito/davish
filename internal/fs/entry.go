package fs

import (
	iofs "io/fs"
	"path"
)

// POSIX file type and special bits, as sent in SFTP attributes.
const (
	sIFMT   = 0o170000
	sIFSOCK = 0o140000
	sIFLNK  = 0o120000
	sIFREG  = 0o100000
	sIFBLK  = 0o060000
	sIFDIR  = 0o040000
	sIFCHR  = 0o020000
	sIFIFO  = 0o010000
	sISUID  = 0o4000
	sISGID  = 0o2000
	sISVTX  = 0o1000
)

func toEntry(p string, fi iofs.FileInfo) Entry {
	e := Entry{
		Name:    path.Base(p),
		Path:    p,
		Size:    fi.Size(),
		ModTime: fi.ModTime().Unix(),
	}
	if st, ok := fileStat(fi); ok {
		e.UID, e.GID = st.UID, st.GID
		e.Perm = st.Mode & 0o7777
		e.Mode = lsMode(st.Mode)
		e.Kind = kindOf(st.Mode)
	} else {
		// Servers always send permissions; this only guards odd ones.
		e.Perm = uint32(fi.Mode().Perm())
		e.Kind, e.Mode = KindFile, lsMode(sIFREG|e.Perm)
		if fi.IsDir() {
			e.Kind, e.Mode = KindDir, lsMode(sIFDIR|e.Perm)
		}
	}
	e.Dir = e.Kind == KindDir
	return e
}

func kindOf(mode uint32) string {
	switch mode & sIFMT {
	case sIFREG:
		return KindFile
	case sIFDIR:
		return KindDir
	case sIFLNK:
		return KindLink
	}
	return KindOther
}

// lsMode renders mode the way ls -l does, including setuid, setgid and
// sticky bits.
func lsMode(mode uint32) string {
	b := []byte("?rwxrwxrwx")
	switch mode & sIFMT {
	case sIFREG:
		b[0] = '-'
	case sIFDIR:
		b[0] = 'd'
	case sIFLNK:
		b[0] = 'l'
	case sIFCHR:
		b[0] = 'c'
	case sIFBLK:
		b[0] = 'b'
	case sIFIFO:
		b[0] = 'p'
	case sIFSOCK:
		b[0] = 's'
	}
	for i := range 9 {
		if mode&(1<<(8-i)) == 0 {
			b[i+1] = '-'
		}
	}
	special := func(bit uint32, at int, set, unset byte) {
		if mode&bit == 0 {
			return
		}
		if b[at] == '-' {
			b[at] = unset
		} else {
			b[at] = set
		}
	}
	special(sISUID, 3, 's', 'S')
	special(sISGID, 6, 's', 'S')
	special(sISVTX, 9, 't', 'T')
	return string(b)
}
