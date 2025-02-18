package pkg

type (
	encoderCodecSetting string
	EncoderOption       = func(*Encoder) error
)

const (
	EncoderCodecNoSetting           encoderCodecSetting = "None"
	EncoderCodecDefaultSetting      encoderCodecSetting = "default"
	EncoderCodecHighQualitySetting  encoderCodecSetting = "high-quality"
	EncoderCodecLowLatencySetting   encoderCodecSetting = "low-latency"
	EncoderCodecLowBandwidthSetting encoderCodecSetting = "low-bandwidth"
)
