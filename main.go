package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/bogem/id3v2/v2"

	tmc "github.com/firepear/thrasher-music-catalog"
	tmcu "github.com/firepear/thrasher-music-catalog/updater"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ver      string
	cat      *tmc.Catalog
	fcreate  bool
	fscan    bool
	fscanall bool
	fadd     bool
	frm      bool
	fquery   bool
	fqquery  bool
	fqrecent bool
	fdebug   bool
	fversion bool
	flimit   int
	foffset  int
	fdbfile  string
	fmusic   string
	ffilter  string
	forder   string
	ftrim    string
	fcutoff  int
	genres   map[int]string
	genreg   *regexp.Regexp
	conf     *tmc.Config
	trks     []string
	id3opts id3v2.Options
)

func init() {
	ver = "v0.6.0"
	// read config file, if it exists
	var err error
	conf, err = tmc.ReadConfig()
	if err != nil {
		fmt.Printf("error reading config: '%s'; continuing with null config...\n", err)
		conf = &tmc.Config{}
	}

	// set up ID3 code
	id3opts = id3v2.Options{Parse: true}

	// handle flags
	flag.BoolVar(&fversion, "v", false, "display version number")
	flag.BoolVar(&fcreate, "c", false, "create new db")
	flag.BoolVar(&fscan, "s", false, "scan for new tracks efficiently")
	flag.BoolVar(&fscanall, "sf", false, "scan, force processing of all dirs")
	flag.BoolVar(&fdebug, "d", false, "print debug info")
	flag.BoolVar(&fadd, "a", false, "add facet to tracks")
	flag.BoolVar(&frm, "r", false, "remove facet from tracks")
	flag.BoolVar(&fquery, "q", false, "query and print track paths")
	flag.BoolVar(&fqquery, "qq", false, "query and print track details")
	flag.BoolVar(&fqrecent, "qr", false, "query and print recent track paths")
	flag.IntVar(&flimit, "l", 0, "query limit (default: size of filter set)")
	flag.IntVar(&foffset, "o", 0, "query offset (default: 0)")
	flag.IntVar(&fcutoff, "co", 0, "track count minimum for artist list inclusion")
	flag.StringVar(&fdbfile, "db", "", "database file to use")
	flag.StringVar(&fmusic, "m", "", "music directory to scan")
	flag.StringVar(&ffilter, "f", "", "filter format string to operate on")
	flag.StringVar(&forder, "ob", "", "comma-delineated list of attributes to order query by")
	flag.StringVar(&ftrim, "t", "", "prefix to remove from track paths")
	flag.Parse()

	// if fdbfile is set, override dbfile
	if fdbfile != "" {
		conf.DbFile = fdbfile
	}
	// ditto musicdir
	if fmusic != "" {
		conf.MusicDir = fmusic
	}
	// and if we still don't have a dbfile, bail
	if conf.DbFile == "" {
		fmt.Println("database file must be specified; see --help")
		os.Exit(1)
	}
	// handle cutoff
	if fcutoff == 0 && conf.ArtistCutoff == 0 {
		// can't both be zero
		fmt.Println("cutoff must be specified; see --help")
		os.Exit(1)
	}
	if conf.ArtistCutoff == 0 {
		// copy arg value into conf if we have one
		conf.ArtistCutoff = fcutoff
	}

	// setup genre stuff
	genreg = regexp.MustCompile("[0-9]+")
	genres = map[int]string{
		0: "Blues", 1: "Classic Rock", 2: "Country", 3: "Dance", 4: "Disco", 5: "Funk",
		6: "Grunge", 7: "Hip-Hop", 8: "Jazz", 9: "Metal", 10: "New Age", 11: "Oldies",
		12: "Other", 13: "Pop", 14: "R&B", 15: "Rap", 16: "Reggae", 17: "Rock",
		18: "Techno", 19: "Industrial", 20: "Alternative", 21: "Ska", 22: "Death Metal",
		23: "Pranks", 24: "Soundtrack", 25: "Euro-Techno", 26: "Ambient", 27: "Trip-Hop",
		28: "Vocal", 29: "Jazz+Funk", 30: "Fusion", 31: "Trance", 32: "Classical",
		33: "Instrumental", 34: "Acid", 35: "House", 36: "Game", 37: "Sound Clip", 38: "Gospel",
		39: "Noise", 40: "AlternRock", 41: "Bass", 42: "Soul", 43: "Punk", 44: "Space",
		45: "Meditative", 46: "Instrumental Pop", 47: "Instrumental Rock", 48: "Ethnic",
		49: "Gothic", 50: "Darkwave", 51: "Techno-Industrial", 52: "Electronic", 53: "Pop-Folk",
		54: "Eurodance", 55: "Dream", 56: "Southern Rock", 57: "Comedy", 58: "Cult",
		59: "Gangsta Rap", 60: "Top 40", 61: "Christian Rap", 62: "Pop / Funk", 63: "Jungle",
		64: "Native American", 65: "Cabaret", 66: "New Wave", 67: "Psychedelic", 68: "Rave",
		69: "Showtunes", 70: "Trailer", 71: "Lo-Fi", 72: "Tribal", 73: "Acid Punk",
		74: "Acid Jazz", 75: "Polka", 76: "Retro", 77: "Musical", 78: "Rock & Roll",
		79: "Hard Rock", 80: "Folk", 81: "Folk-Rock", 82: "National Folk", 83: "Swing",
		84: "Fast Fusion", 85: "Bebob", 86: "Latin", 87: "Revival", 88: "Celtic",
		89: "Bluegrass", 90: "Avantgarde", 91: "Gothic Rock", 92: "Progressive Rock",
		93: "Psychedelic Rock", 94: "Symphonic Rock", 95: "Slow Rock", 96: "Big Band",
		97: "Chorus", 98: "Easy Listening", 99: "Acoustic", 100: "Humour", 101: "Speech",
		102: "Chanson", 103: "Opera", 104: "Chamber Music", 105: "Sonata", 106: "Symphony",
		107: "Booty Bass", 108: "Primus", 109: "Porn Groove", 110: "Satire", 111: "Slow Jam",
		112: "Club", 113: "Tango", 114: "Samba", 115: "Folklore", 116: "Ballad",
		117: "Power Ballad", 118: "Rhythmic Soul", 119: "Freestyle", 120: "Duet",
		121: "Punk Rock", 122: "Drum Solo", 123: "A Cappella", 124: "Euro-House",
		125: "Dance Hall", 126: "Goa", 127: "Drum & Bass", 128: "Club-House", 129: "Hardcore",
		130: "Terror", 131: "Indie", 132: "BritPop", 133: "Negerpunk", 134: "Polsk Punk",
		135: "Beat", 136: "Christian Gangsta Rap", 137: "Heavy Metal", 138: "Black Metal",
		139: "Crossover", 140: "Contemporary Christian", 141: "Christian Rock",
		142: "Merengue", 143: "Salsa", 144: "Thrash Metal", 145: "Anime", 146: "JPop",
		147: "Synthpop", 148: "Abstract", 149: "Art Rock", 150: "Baroque", 151: "Bhangra",
		152: "Big Beat", 153: "Breakbeat", 154: "Chillout", 155: "Downtempo", 156: "Dub",
		157: "EBM", 158: "Eclectic", 159: "Electro", 160: "Electroclash", 161: "Emo",
		162: "Experimental", 163: "Garage", 164: "Global", 165: "IDM", 166: "Illbient",
		167: "Industro-Goth", 168: "Jam Band", 169: "Krautrock", 170: "Leftfield",
		171: "Lounge", 172: "Math Rock", 173: "New Romantic", 174: "Nu-Breakz",
		175: "Post-Punk", 176: "Post-Rock", 177: "Psytrance", 178: "Shoegaze",
		179: "Space Rock", 180: "Trop Rock", 181: "World Music", 182: "Neoclassical",
		183: "Audiobook", 184: "Audio Theatre", 185: "Neue Deutsche Welle",
		186: "Podcast", 187: "Indie Rock", 188: "G-Funk", 189: "Dubstep", 190: "Garage Rock",
		191: "Psybient",
	}
}

func main() {
	if fversion {
		fmt.Println(ver)
		os.Exit(0)
	}

	var err error
	if fdebug {
		fmt.Println("DEBUG> Config")
		fmt.Printf("\tDbFile: %s\n\tMusicDir: %s\n\tArtistCutoff: %d\n",
			conf.DbFile, conf.MusicDir, conf.ArtistCutoff)
	}

	// create an updater instance
	upd, err := tmcu.New(conf.DbFile)
	if err != nil {
		fmt.Printf("error creating updater: %s", err)
		os.Exit(1)
	}
	defer upd.Close()
	// then handle database creation, if asked
	if fcreate {
		// we've been asked to create the db; do so
		err := upd.CreateDB()
		if err != nil {
			fmt.Printf("couldn't create db: %s\n", err)
			os.Exit(2)
		}
		fmt.Printf("database initialized in %s\n", conf.DbFile)
		os.Exit(0)
	}

	// instantiate a catalog instance
	cat, err = tmc.New(conf, "tmctool")
	cat.TrimPrefix = ftrim
	if err != nil {
		fmt.Printf("error creating catalog: %s", err)
		os.Exit(1)
	}
	defer cat.Close()

	// handle setting filter, if we have a format string
	if ffilter != "" {
		err = cat.Filter(ffilter)
		if err != nil {
			fmt.Printf("error parsing filter: %s\n", err)
			os.Exit(3)
		}
		trks = []string{}
		if fdebug {
			fmt.Printf("DEBUG> filter: '%s', %v, %d\n", cat.FltrStr, cat.FltrVals, cat.FltrCount)
		}
	}

	switch {
	case fscan || fscanall:
		// scan for new tracks
		stat, err := os.Stat(conf.MusicDir)
		if err != nil {
			fmt.Printf("can't access musicdir '%s': %s\n", conf.MusicDir, err)
			os.Exit(3)
		}
		if !stat.IsDir() {
			fmt.Printf("%s is not a directory\n", conf.MusicDir)
			os.Exit(3)
		}

		err = scanmp3s(conf, fscanall)
		if err != nil {
			fmt.Printf("error during scan: %s\n", err)
			os.Exit(3)
		}
	case fqrecent:
		// display tracks on recently added albums
		trks, err = cat.QueryRecent()
		if err != nil {
			fmt.Printf("error getting recent tracks: %s\n", err)
			os.Exit(3)
		}
		for _, trk := range trks {
			fmt.Println(trk)
		}
	case fquery || fqquery:
		// query catalog and produce output
		if trks == nil {
			// unless trks isn't set, which means a filter hasn't been set
			fmt.Println("running a query requires a filter to be set; exiting")
			os.Exit(1)
		}
		trks, err = cat.Query(forder, flimit, foffset)
		if err != nil {
			fmt.Printf("error querying catalog: %s\n", err)
			os.Exit(2)
		}
		if fdebug {
			fmt.Printf("DEBUG> query: '%s', %v\n----\n", cat.QueryStr, cat.QueryVals)
		}
		if fqquery {
			// fetch and print track details
			for _, trk := range trks {
				i := cat.TrkInfo(trk)
				if len(i.Artist) > 30 {
					i.Artist = i.Artist[:29] + "…"
				}
				if len(i.Title) > 50 {
					i.Title = i.Title[:49] + "…"
				}
				if len(i.Album) > 30 {
					i.Album = i.Album[:29] + "…"
				}
				fmt.Printf("%3d | %-30s | %-50s | %-30s | %d |\t%s\n",
					i.Num, i.Artist, i.Title, i.Album, i.Year, i.Facets)
			}
		} else {
			// just print the track paths
			for _, trk := range trks {
				fmt.Println(trk)
			}
		}
	default:
		fmt.Println("No op requested")
	}
}
