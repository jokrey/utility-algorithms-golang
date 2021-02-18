package stringencoder

import (
	"errors"
	"strconv"
	"strings"
)

/// Allows for a simple data to be encoded into a String.
/// Nesting this allows for infinitly complex storage structures.
///      Nesting is achieved by letting complex classes encode their raw data using this utility class
///         (good practise is to implement EncodableAsString interface and provide a constructor that takes an encoded String)
/// Accessing the data can be done over tags. If a Tag doesn't exist, the method(getEntry) will return null.
/// When switching around between versions this has to be caught, but if handled right, the remaining data is not lost.
/// The resulting string can then be stored without much effort.
///
/// NOTE: This is ENCODING, NOT ENCRYPTING
///    Though the results of this method may be hard to read for a human, should the data be sensitive encryption is still required.

type AdvancedStringEncoder struct {
	work_in_progress string
}

func New_AdvancedStringEncoder(initiate_str string) AdvancedStringEncoder {
	return AdvancedStringEncoder{initiate_str}
}

func (ase AdvancedStringEncoder) SetWorkingString(wip string) {
	ase.work_in_progress = wip
}
func (ase AdvancedStringEncoder) GetEncodedString() string {
	return ase.work_in_progress
}


func (ase *AdvancedStringEncoder) AddEntry(tag, entry string) {
	ase.DeleteEntry(tag)
	ase.work_in_progress = ase.work_in_progress + li_encode_single(tag)+li_encode_single(entry)
}
func (ase AdvancedStringEncoder) GetEntry(tag string) (string, error) {
	current_i:=0
	for { //<=> to do-while
		tag_content_pair := li_decode_multiple_of_until(ase.work_in_progress, current_i, 2)
		if len(tag_content_pair)!=2 {
			return "", errors.New("internal li error")
		}
		if tag_content_pair[0] == tag {
			return tag_content_pair[1], nil
		}
		current_i = ase.readto_next_entry(current_i)
		if current_i==0 {break}
	}
	return "", errors.New("no entry with tag "+tag+" found")
}
func (ase *AdvancedStringEncoder) DeleteEntry(tag string) (string, error) {
	current_i:=0
	for {
		tag_content_pair := li_decode_multiple_of_until(ase.work_in_progress, current_i, 2)
		if len(tag_content_pair)!=2 {
			return "", errors.New("internal li error")
		}
		if tag_content_pair[0] == tag {
			startIndex_endIndex_of_TAG := get_startAndEndIndexOf_NextLIString(current_i, ase.work_in_progress)
			startIndex_endIndex_of_CONTENT := get_startAndEndIndexOf_NextLIString(startIndex_endIndex_of_TAG[1], ase.work_in_progress)
			ase.work_in_progress = ase.work_in_progress[:current_i] + ase.work_in_progress[startIndex_endIndex_of_CONTENT[1]:] //NOT TESTED WELL
			return tag_content_pair[1], nil
		}
		current_i = ase.readto_next_entry(current_i)
		if current_i==0 {break}
	}
	return "", errors.New("no entry with tag "+tag+" found")
}
func (ase AdvancedStringEncoder) GetEntryNoErr(tag string) string {
	entry, err :=ase.GetEntry(tag)
	if err == nil {
		return entry
	} else {
		return ""
	}
}
func (ase *AdvancedStringEncoder) DeleteEntryNoErr(tag string) string {
	entry, err :=ase.DeleteEntry(tag)
	if err == nil {
		return entry
	} else {
		return ""
	}
}
func (ase AdvancedStringEncoder) readto_next_entry(current_i int) int {
	current_i = get_startAndEndIndexOf_NextLIString(current_i, ase.work_in_progress)[1] //skip tag
	current_i = get_startAndEndIndexOf_NextLIString(current_i, ase.work_in_progress)[1] //skip content
	if current_i>=len(ase.work_in_progress) {
		current_i = 0
	}
	return current_i
}



///boolean shorts
func (ase *AdvancedStringEncoder) AddEntry_bool(tag string, b bool) {
	if b {
		ase.AddEntry(tag, "t")
	} else {
		ase.AddEntry(tag, "f")
	}
}
func (ase AdvancedStringEncoder) GetEntry_bool(tag string) bool {
	return ase.GetEntryNoErr(tag) == "t"
}
func (ase *AdvancedStringEncoder) DeleteEntry_bool(tag string) bool {
	return ase.DeleteEntryNoErr(tag) == "t"
}

///int64 shorts
func (ase *AdvancedStringEncoder) AddEntry_i64(tag string, b int64) {
	ase.AddEntry(tag, strconv.FormatInt(b, 10))
}
func (ase AdvancedStringEncoder) GetEntry_i64(tag string) int64 {
	entry_i64, err := strconv.ParseInt(ase.GetEntryNoErr(tag), 10, 0)
	if err==nil {
		return entry_i64
	} else {
		return 0
	}
}
func (ase *AdvancedStringEncoder) DeleteEntry_i64(tag string) int64 {
	entry_i64, err := strconv.ParseInt(ase.DeleteEntryNoErr(tag), 10, 0)
	if err==nil {
		return entry_i64
	} else {
		return 0
	}
}

///EncodableAsString
func (ase *AdvancedStringEncoder) AddEntry_encodable(tag string, encodableObject EncodableAsString) {
	ase.AddEntry(tag, encodableObject.GetEncodedString());
}
func (ase AdvancedStringEncoder) NewEncodableFromEntry(tag string, dummy EncodableAsString) {
	entry, err := ase.GetEntry(tag);
	if err == nil {
		dummy.NewFromEncodedString(entry);
	}
}





////LI-Encoding
////Length-Indicator based encoding
//TODO the + operator for string may be slow, replace with sometihng else
func li_encode_multiple(strings []string) string {
	encode_builder := ""
	for _, str:= range strings {
		encode_builder= encode_builder + li_encode_single(str)
	}
	return encode_builder
}
func li_decode_all(encoded_str string) []string {
	return li_decode_multiple_of_until(encoded_str, 0, -1)
}
func li_decode_multiple_of_until(encoded_str string, start_index, limit int) []string {
	decoded_builder := make([]string, 0)

	startIndex_endIndex_ofNextLIString := []int{0, start_index}
	for len(startIndex_endIndex_ofNextLIString) == 2 {
		if limit>-1 && len(decoded_builder)>=limit { //-1 because after while one more is added added
			break
		}
		startIndex_endIndex_ofNextLIString = get_startAndEndIndexOf_NextLIString(startIndex_endIndex_ofNextLIString[1], encoded_str)
		if len(startIndex_endIndex_ofNextLIString)==2 {
			decoded_builder = append(decoded_builder, encoded_str[startIndex_endIndex_ofNextLIString[0] : startIndex_endIndex_ofNextLIString[1]])
		}
	}

	return decoded_builder
}
//TODO the + operator for string may be slow, replace with sometihng else
func li_encode_single(str string) string {
	return getLengthIndicatorFor(str)+str
}
func Li_decode_single(encoded_str string) string {
	startIndex_endIndex_ofNextLIString := get_startAndEndIndexOf_NextLIString(0, encoded_str)
	if len(startIndex_endIndex_ofNextLIString)==2 {
		return encoded_str[startIndex_endIndex_ofNextLIString[0] : startIndex_endIndex_ofNextLIString[0]+startIndex_endIndex_ofNextLIString[1]]
	} else {
		return ""
	}
}

func getLengthIndicatorFor(str string) string {
	lengthIndicatorBuilder := make([]string, 1)
	lengthIndicatorBuilder[0] = strconv.Itoa(len(str) + 1) //Attention: The +1 is necessary because we are adding a pseudo random char to the beginning of the splitted char to hinder a bug, if somechooses to only save a medium sized int. (It would be interpreted as a lengthIndicator)
	for len(lengthIndicatorBuilder[0]) != 1 {
		lengthIndicatorBuilder = append([]string{strconv.Itoa( len(lengthIndicatorBuilder[0]) )}, lengthIndicatorBuilder...) //prepend
	}
	return strings.Join(lengthIndicatorBuilder[:],"")+GetPseudoRandomHashedCharAsString(str)
}

func get_startAndEndIndexOf_NextLIString(start_index int, str string) []int {
	var i int = start_index
	if i+1>len(str) {
		return []int {}
	}
	lengthIndicator := str[i : i+1]
	i+=1
	for {
		lengthIndicator_asInt := getInt(lengthIndicator, -1)
		if i+lengthIndicator_asInt>len(str) {
			return []int{}
		}
		eitherDataOrIndicator :=  str[i : i+lengthIndicator_asInt]
		ifitwasAnIndicator := getInt(eitherDataOrIndicator, -1)
		if ifitwasAnIndicator > lengthIndicator_asInt && i+ifitwasAnIndicator <= len(str) {
			i+=lengthIndicator_asInt
			lengthIndicator=eitherDataOrIndicator
		} else {
			if lengthIndicator_asInt == -1 {
				return []int {}
			} else {
				return []int {i+1, i+lengthIndicator_asInt} //i+1 for the pseudo random hash char
			}
		}
	}
}




func getInt(s string, backup int) int {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i
	} else {
		return backup
	}
}

func GetPseudoRandomHashedCharAsString(from string) string {
	possibleChars := "abcdefghijklmnopqrstuvwxyz!?()[]{}=-+*#"
	additionHashSaltThingy := 0
	for _, b := range []byte(from) {
		additionHashSaltThingy += int(b)
	}
	return string([]rune(possibleChars)[(len(from)+additionHashSaltThingy) % len(possibleChars)])
}