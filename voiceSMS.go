package main

import (
        "bytes"
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "net/smtp"
        "crypto/tls"
        "encoding/base64"
        "mime/multipart"
        "net/textproto"
        "path/filepath"
        "os"
        "os/exec"
        //"math/rand"
        "strings"
        "time"
        "regexp"
        "github.com/CyCoreSystems/ari/v6"
        "github.com/CyCoreSystems/ari/v6/client/native"
        "github.com/CyCoreSystems/ari/v6/ext/play"
        "github.com/CyCoreSystems/ari/v6/ext/record"
        "github.com/openai/openai-go"
        "github.com/openai/openai-go/option"
        openaiss "github.com/sashabaranov/go-openai"
)

////////////////////////////////////////////////////
//// begin global constants & variables ////////////

// // define constants ////

// // default language for transcription, etc
const defLang = "english"

// // Telnyx SMS URL
const telnyxUrl = "https://api.telnyx.com/v2/messages"
const telnyxUrlGroup = "https://api.telnyx.com/v2/messages/group_mms"

// // default implicit recordings directory
const recordingDir = "/opt/pbxware/pw/var/spool/asterisk/recording/"

// // minimum arguments needed
const minArgs = 5

// // maximum silence time for recording in seconds
const maxRecordingSilence = 3

// // maximum recording time in seconds
const maxRecordingTime = 40

// // when doing multiple instances with different command-line arguments, the env value does not always seem to be passed, so add explicitly as below
const oaiKey = "YOUR_OPENAI_KEY"

// // when doing multiple instances with different command-line arguments, the env value does not always seem to be passed, so add explicitly as below
const telnyxKey = "YOUR_TELNYX_KEY"

// // constants for openAI Transcription errors
const noTranscription = "No transcription"
const noAudio = "No audio"

// SMTP server configuration
const smtpHost = "mail.server.com"
const smtpPort = "465"
const smtpUsername = "user@server.com"
const smtpPassword = "PaSsw0rd"
const smtpFrom = "sender@server.com"

// //
const  noCallbackNum = "nocallbacknum"

// // define global variables that cannot be constants since generated form env variables

// // define variables to be replaced by command-line arguments
var ariIP string                                //// IP address that the ARI will be listened for on
var ariUser string                              //// ARI Username
var ariPass string                              //// ARI Password
var ariAppName string                           //// ARI app name
var ariTenant string                            //// Tenant number (1 if Call Centre or Business Edition)
var SMSName string = "ARI Application"          //// Name to add to the sent SMS
var SMSTo = "+15555555555"                      //// a valid e.164 formatted SMS DID (or comma-separated list)
var SMSFrom = "+15555555556"                    //// a valid e.164 formatted SMS DID associated with the SMS profile
var allowCallback = "callbacknum"               //// anything other than matching noCallbackNum will be considered as callbacknum
var ariLang = defLang                           //// language to translate to if that option is chosen
var ariStyle = "transcribe"                     //// transcribe, translate, transback, qanda or chat

var ariVoice = "-"

var emailTo = "-"                               //// address to send email to with attachment

var logfile = "/path/to/your/logs/arilog.txt" //// need to arrange log rotation

// // Sound file variables

// // DMTF instructions filename
var dtmfInstructions = "greeting-enter_pound_or_number"         //// may be replaced at command line, file must exist : recording prompt to enter # or callback number+#

// // Recording instructions filename
var recordingInstructions = "greeting-message_after_beep"       //// may be replaced at command line, file must exist : recording prompt to record message and press pound or hangup

// // Goodbye filename
var goodbyeSound = "greeting-thankyou"          //// may be replaced at command line, file must exist : recording to terminate the app if caller has not already hung up

// // verbosity = "verbose" : log lines will be echoed to stdout / any other value : lines will be logged only
var verbosity = "silent"        //// may be replaced at command line, anything other than verbose will be considered silent

// // end variables to be replaced by command-line arguments
///////////////////////////////

// // recording subdirectory, appended to recordingDir = "/opt/pbxware/pw/var/spool/asterisk/recording/"
// // to make managing the recording files simpler : find . . . -exec rm {} \;
var recordingSubDir = "ari"

// // name of the executable file : to be replaced at runtime
var myName = "executable"

var lang = ariLang

//// choose a default voice
var rvoice = openai.AudioSpeechNewParamsVoiceShimmer

//// end global constants & variables ////////////
//////////////////////////////////////////////////

func main() {

        iargs := os.Args

        nameParts := strings.Split(iargs[0],"/")
        myName = nameParts[len(nameParts)-1]

        //// if there are enough (minArgs) arguments, assign them and proceed
        if len(iargs) >= minArgs+1 {

                ariIP = iargs[1]
                ariUser = iargs[2]
                ariPass = iargs[3]
                ariAppName = iargs[4]
                ariTenant = iargs[5]

                //// if there are more args, assign them to the appropriate globals
                if len(iargs) > minArgs+1 {
                        for i := minArgs + 1; i <= len(iargs)-1; i++ {
                                if iargs[i] != "-" {
                                        switch i {
                                        case 6:
                                                SMSName = iargs[i]
                                        case 7:
                                                SMSTo = iargs[i]
                                        case 8:
                                                SMSFrom = iargs[i]
                                        case 9:
                                                allowCallback = iargs[i]
                                        case 10:
                                                ariLang = iargs[i]
                                        case 11:
                                                ariStyle = iargs[i]
                                        case 12:
                                                ariVoice = iargs[i]
                                        case 13:
                                                emailTo = iargs[i]
                                        case 14:
                                                logfile = iargs[i]
                                        case 15:
                                                dtmfInstructions = iargs[i]
                                        case 16:
                                                recordingInstructions = iargs[i]
                                        case 17:
                                                goodbyeSound = iargs[i]
                                        case 18:
                                                verbosity = iargs[i]
                                        }
                                }
                        }
                }


        //// some common abbreviations, but we can also supply the full name at command-line
        switch ariLang {
                case "-":
                        lang = defLang
                case "en":
                        lang = "english"
                case "fr":
                        lang = "french"
                case "es":
                        lang = "spanish"
                case "de":
                        lang = "german"
                case "pt" :
                        lang = "portuguese"
                case "it":
                        lang = "italian"
                case "gk":
                        lang = "greek"
                case "af":
                        lang = "afrikaans"
                case "sq":
                        lang = "albanian"
                case "eu":
                        lang = "basque"
                case "bg":
                        lang = "bulgarian"
                case "ca":
                        lang = "catalan"
                case "hr":
                        lang = "croatian"
                case "cs":
                        lang = "czech"
                case "da":
                        lang = "danish"
                case "nl":
                        lang = "dutch"
                case "et":
                        lang = "estonian"
                case "fi":
                        lang = "finnish"
                case "hu":
                        lang = "hungarian"
                case "is":
                        lang = "icelandic"
                case "in":
                        lang = "indonesian"
                case "lv":
                        lang = "latvian"
                case "lt":
                        lang = "lithuanian"
                case "mk":
                        lang = "macedonian"
                case "no":
                        lang = "norwegian"
                case "pl" :
                        lang = "polish"
                case "ro":
                        lang = "romanian"
                case "ru":
                        lang = "russian"
                case "sr":
                        lang = "serbian"
                case "sl":
                        lang = "slovak"
                case "sk":
                        lang = "slovenian"
                case "sv":
                        lang = "swedish"
                case "th":
                        lang = "turkish"
                case "vi":
                        lang = "vietnamese"
                case "hy":
                        lang = "armenian"
                case "he":
                        lang = "hebrew"
                case "ja":
                        lang = "japanese"
                case "ko":
                        lang = "korean"
                case "zh":
                        lang = "chinese"
                case "tl":
                        lang = "tagalog"
                case "ms":
                        lang = "malay"
                case "tr":
                        lang = "thai"
                case "ar":
                        lang = "arabic"
                case "ps":
                        lang = "pashto"
                case "hi":
                        lang = "hindi"
                case "fa":
                        lang = "farsi"
                case "ur":
                        lang = "urdu"
                case "sw":
                        lang = "swahili"
                case "ki":
                        lang = "kikongo"
                case "ig":
                        lang = "igbo"
                case "mw":
                        lang = "mohawk"
                default:
                        lang = defLang

        }


    //// option to define a particular voice
    //// figure out why Ash, Coral & Sage are invalid here, while they work fine in another script on the same system . . .

    switch strings.ToLower(ariVoice) {
        case "-":
            rvoice = openai.AudioSpeechNewParamsVoiceShimmer
        case "shimmer":
            rvoice = openai.AudioSpeechNewParamsVoiceShimmer
        case "alloy":
            rvoice = openai.AudioSpeechNewParamsVoiceAlloy
        //case "ash":
        //    rvoice = openai.AudioSpeechNewParamsVoiceAsh
        //case "coral":
        //    rvoice = openai.AudioSpeechNewParamsVoiceCoral
        case "echo":
            rvoice = openai.AudioSpeechNewParamsVoiceEcho
        case "Fable":
            rvoice = openai.AudioSpeechNewParamsVoiceFable
        case "onyx":
            rvoice = openai.AudioSpeechNewParamsVoiceOnyx
        case "Nova":
            rvoice = openai.AudioSpeechNewParamsVoiceNova
        //case "Sage":
        //    rvoice = openai.AudioSpeechNewParamsVoiceSage
        default:
            rvoice = openai.AudioSpeechNewParamsVoiceShimmer
    }


                //// prepare the context
                cntxt, cancel := context.WithCancel(context.Background())
                defer cancel()

                logit("Connecting to ARI", "info", "")

                //// prepare the ARI client
                ariclient, err := native.Connect(&native.Options{
                        Application:  ariAppName,
                        Username:     ariUser,
                        Password:     ariPass,
                        URL:          fmt.Sprintf("http://%s:8088/ari", ariIP),
                        WebsocketURL: fmt.Sprintf("ws://%s:8088/ari/events", ariIP),
                })
                if err != nil {
                        logit("Failed to build native ARI client", "error", fmt.Sprintf("%v", err))
                        return
                }

                logit("Listening for calls", "info", "")

                //// subscribe to StasisStart events
                sub := ariclient.Bus().Subscribe(nil, "StasisStart")

                //// wait for 'StasisStart' events, (incoming call)
                for {
                        select {
                        case e := <-sub.Events():
                                v := e.(*ari.StasisStart)

                                // get CID Info, etc
                                ChanID := v.Channel.ID
                                ChanName := v.Channel.Name
                                CIDNumber := strings.TrimLeft(v.Channel.Caller.Number, "+") //// remove a potential leading + from CIDNumber
                                CIDName := v.Channel.Caller.Name

                                logit("Received StasisStart", "channel", fmt.Sprintf("%v %v - %v <%v>", ChanID, ChanName, CIDName, CIDNumber))

                                //// handle the individual call
                                go handleCall(cntxt, ariclient.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID)), CIDNumber, CIDName, ChanName, ariTenant)

                        case <-cntxt.Done():
                                return
                        }
                }

        } else { //// not enough arguments supplied, show usage
                usage(myName, fmt.Sprintf("Insufficient Arguments, needs at least %v", minArgs))
        }
}


func getGPTChatss(cntxt context.Context, h *ari.ChannelHandle, channame string, question string, lng string, cname string, cid string, tenant string) (string, error) {

        // added translation request to the question

        qtosend := fmt.Sprintf("Please translate to %v : %v", lng, question)
        recName := ""

        // Create client
        client := openaiss.NewClient(oaiKey)

        if(ariStyle == "chat") {
                qtosend = fmt.Sprintf("Please reply to the following question in the language it is asked, if possible, if not, reply in %v : %v", lng, question)

                messages := make([]openaiss.ChatCompletionMessage,0)

                messages = append(messages, openaiss.ChatCompletionMessage{
                        Role:    openaiss.ChatMessageRoleUser,
                        Content: fmt.Sprintf("you are a helpful canadian assistant, please reply in the language you are asked the question in. If unable, please reply in %v.",lng),
                })

                roundr := 0
                //var userInput string

                for {

                        roundr++
                        if(roundr == 1) {
                                qtosend = question
                        //} else if (roundr == 2){
                        //      //// temporary debug
                        //      qtosend = "What is the weather like there in February ?"
                        } else {
                                //qtosend = "Bye"
                                //// get the next client input
                                //// use handleCall
                                //recordTranscribeReturn(cntxt context.Context, h *ari.ChannelHandle, cname string, cid string, tenant)
                                qtosend, recName = recordTranscribeReturn(cntxt, h, channame, cname, cid, tenant, "chat")
                                logit("Recording","info",fmt.Sprintf("%v",recName))
                        }

                        logit(qtosend,"tmp",fmt.Sprintf("Round : %v",roundr))

                        //if(strings.ToLower(qtosend) == "bye.") {
                        if((strings.ToLower(qtosend) == "bye.") || (strings.HasSuffix(strings.ToLower(qtosend), " bye."))) {
                                break
                        }

                        //// transcribe and send to chatGPT

                        //qtosend =

                        messages = append(messages, openaiss.ChatCompletionMessage{
                                Role:    openaiss.ChatMessageRoleUser,
                                Content: qtosend,
                        })


                        resp, err := client.CreateChatCompletion(
                                context.Background(),
                                //cntxt
                                openaiss.ChatCompletionRequest{
                                        Model: openaiss.GPT3Dot5Turbo,
                                        Messages: messages,
                                        Temperature: 0.7,
                                        MaxTokens:   500,
                                },
                        )
                        if err != nil {
                                logit("","",fmt.Sprintf("Chat error : %v",err))
                                continue
                                //break
                        }



                        assistantResponse := resp.Choices[0].Message.Content

                        /// get and play the response here

                        if err := textToSpeech(cntxt, h, assistantResponse); err != nil {
                                logit("TTS failed", "error", fmt.Sprintf("%v", err))
                                break
                        }

                        messages = append(messages, openaiss.ChatCompletionMessage{
                                Role:    openaiss.ChatMessageRoleAssistant,
                                Content: assistantResponse,
                        })

                }

                transcriptToSend := "Transcription :"

                for i := 0;i<len(messages);i++ {
                        //logit("","",fmt.Sprintf("Message : ","tmp",messages[i]))
                        transcriptToSend += fmt.Sprintf("\n\n%v",messages[i])
                        //logit("Message","tmp",fmt.Sprintf("%v",messages[i]))
                }

                if(emailTo != "-") {
                        // email the transcript

                        mailto := strings.Split(emailTo,",")
                        subject := fmt.Sprintf("Chat transcript - %v <%v>",cname,cid)
                        body := transcriptToSend
                       // attachpath := fmt.Sprintf("%v%v.wav", recordingDir, recName)
                        attachpath := ""

                        //SendEmail(mailto,subject,body,attachpath)
                        err := SendEmail(mailto,subject,body,attachpath)
                        if err != nil {
                                logit("Failed to send email","error",fmt.Sprintf("%v",err))
                                return "", nil
                        }


                }

                return "Bye", nil
                //return "", nil
        } else {

                if(ariStyle == "qanda") {
                        qtosend = fmt.Sprintf("Please reply to the following question in %v : %v", lng, question)
                } else {
                        qtosend = fmt.Sprintf("Please translate to %v : %v", lng, question)
                }
                // Create client
                //client := openaiss.NewClient(oaiKey)

                // Create request
                resp, err := client.CreateChatCompletion(
                        context.Background(),
                        openaiss.ChatCompletionRequest{
                        Model: openaiss.GPT3Dot5Turbo, // or openai.GPT4 for GPT-4
                        Messages: []openaiss.ChatCompletionMessage{
                                {
                                        Role:    openaiss.ChatMessageRoleUser,
                                        Content: qtosend,
                                        },
                                },
                                Temperature: 0.7,
                                MaxTokens:   500,
                },
                )

                if err != nil {
                        return "", fmt.Errorf("ChatCompletion error: %v", err)
                }
        return resp.Choices[0].Message.Content, nil
        }
}


//////// Main call handling function
func handleCall(cntxt context.Context, h *ari.ChannelHandle, cid string, cname string, channame string, tenant string) {

        defer h.Hangup()

        cntxt, cancel := context.WithCancel(cntxt)
        defer cancel()

        end := h.Subscribe(ari.Events.StasisEnd)
        defer end.Cancel()

        go func() {
                <-end.Events()
                //// removed cancel()
                //// so we don't pull the context from secondary functions
                //cancel()
        }()

        if err := h.Answer(); err != nil {
                logit("Failed to answer", "error", fmt.Sprintf("%v", err))
                return
        }

        //logit("New call started", "channel", fmt.Sprintf("%v", h.ID()))

        var callBack string
        var newNum = "#"

        //// if allowCallback has not been set to match noCallbackNum, get an alternat callback number
        if allowCallback != noCallbackNum {
                newNum = getDTMF(cntxt, h, tenant)
        } else {
                //// pause so the 'Record you message . . .' is not clipped at the beginning
                time.Sleep(1*time.Second)
        }

        //// if a new number was entered, copy it to callBack, otherwise keep the CID
        if newNum == "#" {
                callBack = cid
        } else {
                callBack = strings.TrimRight(newNum, "#")
        }

        transcript, recName := recordTranscribeReturn(cntxt, h, channame, cname, cid, tenant, "main")

        //logit("Recorded","info",fmt.Sprintf("%v\n%v",recName, transcript))

        //// only send SMS, or perform other action if there is a transcription
        if (transcript != noTranscription) && (transcript != noAudio) {

                if((ariStyle == "qanda") || (ariStyle == "chat")) {
                        logit("Starting GPT Chat","info",transcript)

                        langTo := parseLang(lang, cname)

                        chatresp, err := getGPTChatss(cntxt, h, channame, transcript, langTo, cname, cid, tenant)
                        if err != nil {
                                logit("No ChatGPT response","error",fmt.Sprintf("%v",err))
                        }

                        //logit(chatresp,"tmp","")


                        if err := textToSpeech(cntxt, h, chatresp); err != nil {
                                logit("TTS failed", "error", fmt.Sprintf("%v", err))
                        }

                //} else if(ariStyle == 'chat') {

                } else if(ariStyle != "transback") {
                //if(ariStyle != "transback") {
                        //// only show thw callback line if it is needed (allowCallback != noCallbackNum)
                        var callBackLine = ""
                        if allowCallback != noCallbackNum {
                                callBackLine = fmt.Sprintf("\nCallback :%v",callBack)
                        }
                        //// prepare the transcription to be sent as SMS, adding Date/Time, Caller ID, CallBack Number and the AI disclaimer
                        smsMessage := fmt.Sprintf("%v\n\n%v\n%v <%v>%v\n\n%v\n\n**Transcribed by openAI**", SMSName, strings.Split(fmt.Sprintf("%v", time.Now()), ".")[0], cname, cid, callBackLine, transcript)
                        //// Send transcription as SMS

                        //// Send SMS as a native go function
                        if strings.Contains(SMSTo,",") {
                                telnyxSendGroup(smsMessage)
                        } else {
                                telnyxSendIndiv(smsMessage)
                        }


                        if(emailTo != "-") {
                                // send email
                                mailto := strings.Split(emailTo,",")
                                subject := fmt.Sprintf("VM Message - %v <%v>",cname,cid)
                                body := transcript
                                attachpath := fmt.Sprintf("%v%v.wav", recordingDir, recName)

                                //SendEmail(mailto,subject,body,attachpath)
                                err := SendEmail(mailto,subject,body,attachpath)
                                if err != nil {
                                        logit("Failed to send email","error",fmt.Sprintf("%v",err))
                                        return
                                }
                        }
                } else {

                        if err := textToSpeech(cntxt, h, transcript); err != nil {
                              logit("TTS failed", "error", fmt.Sprintf("%v", err))
                        }
                }



        } else {
                logit("No transcription, SMS will not be sent","error",transcript)
        }

        //// directory qualifier for Multi-Tenant (ex '202/' for a file in Tenant 202's custom directory)
        xdir := fmt.Sprintf("%v/", tenant)
        //// if tenant = 1, it is a Call Centre or Business Stand-alone . . . so no Tenant
        if tenant == "1" {
                xdir = ""
        }


        if err := play.Play(cntxt, h, play.URI(fmt.Sprintf("sound:%v%v", xdir, goodbyeSound))).Err(); err != nil {
                logit("Failed to play sound file", "error", fmt.Sprintf("%v", err))
                return
        }

        logit("Call completed", "info", fmt.Sprintf("%v", h.ID()))
}

//////// Secondary functions

func parseLang(deflang string, incname string) string {

        lrtrn := deflang

        if(strings.HasPrefix(incname,"lang=")) { ////check for the lang= prefix in the imcoming CID name, as lang=Serbo-Croatian
                langpart := strings.Split(incname,"=")[1]
                reg := regexp.MustCompile("^[a-zA-Z-]+$")
                if(reg.MatchString(langpart)) {
                        lrtrn = langpart
                }
        }

        return lrtrn
}

func SendEmail(to []string, subject string, body string, attachmentPath string) error {

        // Create a new buffer to write the MIME message
        buffer := bytes.NewBuffer(nil)
        writer := multipart.NewWriter(buffer)

        // Set email headers
        headers := make(map[string]string)
        headers["From"] = smtpFrom
        headers["To"] = strings.Join(to, ",")
        headers["Subject"] = subject
        headers["MIME-Version"] = "1.0"
        headers["Content-Type"] = "multipart/mixed; boundary=" + writer.Boundary()

        // Write headers to buffer
        for key, value := range headers {
                buffer.WriteString(fmt.Sprintf("%s: %s\n", key, value))
        }
        buffer.WriteString("\n")

        // Add text part
        textPart, err := writer.CreatePart(textproto.MIMEHeader{
                "Content-Type": []string{"text/plain; charset=utf-8"},
        })
        if err != nil {
                return fmt.Errorf("failed to create text part: %v", err)
        }
        _, err = textPart.Write([]byte(body))
        if err != nil {
                return fmt.Errorf("failed to write text part: %v", err)
        }

        // Add attachment if path is provided
        if attachmentPath != "" {
                file, err := os.Open(attachmentPath)
                if err != nil {
                        return fmt.Errorf("failed to open attachment: %v", err)
                }
                defer file.Close()

                // Get the filename from the path
                fileName := filepath.Base(attachmentPath)

                attachmentPart, err := writer.CreatePart(textproto.MIMEHeader{
                        "Content-Type":              []string{"application/octet-stream"},
                        "Content-Transfer-Encoding": []string{"base64"},
                        "Content-Disposition":       []string{fmt.Sprintf(`attachment; filename="%s"`, fileName)},
                })
                if err != nil {
                        return fmt.Errorf("failed to create attachment part: %v", err)
                }

                // Create base64 encoder
                encoder := base64.NewEncoder(base64.StdEncoding, attachmentPart)
                _, err = io.Copy(encoder, file)
                if err != nil {
                        return fmt.Errorf("failed to encode attachment: %v", err)
                }
                encoder.Close()
        }

        writer.Close()

        // Create SSL connection
        tlsConfig := &tls.Config{
                ServerName: smtpHost,
                MinVersion: tls.VersionTLS12,
        }

        // Connect to SMTP server with SSL
        conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsConfig)
        if err != nil {
                return fmt.Errorf("failed to create SSL connection: %v", err)
        }
        defer conn.Close()

        // Create new SMTP client
        client, err := smtp.NewClient(conn, smtpHost)
        if err != nil {
                return fmt.Errorf("failed to create SMTP client: %v", err)
        }
        defer client.Close()

        // Connect to SMTP server
        auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
        if err = client.Auth(auth); err != nil {
                return fmt.Errorf("failed to authenticate: %v", err)
        }

        // Set sender
        if err = client.Mail(smtpFrom); err != nil {
                return fmt.Errorf("failed to set sender: %v", err)
        }

        // Add recipients
        for _, recipient := range to {
                if err = client.Rcpt(recipient); err != nil {
                        return fmt.Errorf("failed to add recipient %s: %v", recipient, err)
                }
        }

        // Send the email body
        dataWriter, err := client.Data()
        if err != nil {
                return fmt.Errorf("failed to create data writer: %v", err)
        }
        _, err = dataWriter.Write(buffer.Bytes())
        if err != nil {
                return fmt.Errorf("failed to write email body: %v", err)
        }
        err = dataWriter.Close()
        if err != nil {
                return fmt.Errorf("failed to close data writer: %v", err)
        }

        return nil
}


func recordIt(cntxt context.Context, h *ari.ChannelHandle, recname string, tenant string, from string) {

        //// directory qualifier for Multi-Tenant (ex '202/' for a file in Tenant 202's custom directory)
        xdir := fmt.Sprintf("%v/", tenant)
        //// if tenant = 1, it is a Call Centre or Business Stand-alone . . . so no Tenant
        if tenant == "1" {
                xdir = ""
        }

        if err := play.Play(cntxt, h, play.URI(fmt.Sprintf("sound:%v%v", xdir, recordingInstructions))).Err(); err != nil {
                logit("Failed to play file", "error", fmt.Sprintf("%v", err))
                return
        }

        if(from != "chat") {
                //// pause before the beep
                time.Sleep(500 * time.Millisecond)
        }

        beepr := record.Beep()

        rslt, err := record.Record(cntxt, h, record.Format("wav"), record.MaxDuration(maxRecordingTime*time.Second), record.MaxSilence(maxRecordingSilence*time.Second), record.IfExists("overwrite"), beepr, record.TerminateOn("#")).Result()

        if err != nil {
                logit("Failed to record", "error", fmt.Sprintf("%v", err))
                return
        }

        //// save recording to file
        if err := rslt.Save(fmt.Sprintf("%v", recname)); err != nil {
                logit(fmt.Sprintf("Failed to record %v.wav", recname), "error", "")
        } else {
                logit(fmt.Sprintf("Recorded %v.wav", recname), "info", "")
        }
}

// // used in develpment only, reads back supplied digits
func readDigits(cntxt context.Context, h *ari.ChannelHandle, cid string) {

        logit("Reading Digits", "channel", fmt.Sprintf("%v - %v", cid, h.ID()))

        digitList := strings.Split(cid, "")
        for i := 0; i < len(digitList); i++ {
                if err := play.Play(cntxt, h, play.URI(fmt.Sprintf("digits:%v", digitList[i]))).Err(); err != nil {
                        logit(fmt.Sprintf("Failed to play digit file : %v", digitList[i]), "error", fmt.Sprintf("%v", err))
                        return
                }
        }
}

func getDTMF(cntxt context.Context, h *ari.ChannelHandle, tenant string) string {

        //// directory qualifier for Multi-Tenant (ex '202/' for a file in Tenant 202's custom directory)
        xdir := fmt.Sprintf("%v/", tenant)
        //// if tenant = 1, it is a Call Centre or Business Stand-alone . . . so no Tenant
        if tenant == "1" {
                xdir = ""
        }

        dtmfrtrn := ""

        dtmf := h.Subscribe(ari.Events.ChannelDtmfReceived)

        if err := play.Play(cntxt, h, play.URI(fmt.Sprintf("sound:%v%v", xdir, dtmfInstructions))).Err(); err != nil {
                logit("Failed to play file", "error", fmt.Sprintf("%v", err))
                return "#"
        }

        for {
                select {
                case event := <-dtmf.Events():
                        ev := event.(*ari.ChannelDtmfReceived)
                        //// add the dtmf value to the return value
                        dtmfrtrn += ev.Digit
                        //logit(ev.Digit, "dtmf", dtmfrtrn)
                        if ev.Digit == "#" {
                                logit(ev.Digit, "dtmf", dtmfrtrn)
                                return dtmfrtrn
                        } else if len(dtmfrtrn) > 11 {
                                dtmfrtrn += "#"
                                return dtmfrtrn
                        }
                //// avoid no '#' entry, getting us stuck in this loop, limit to 4 seconds run time with no dtmf entry
                case <-time.After(4 * time.Second):
                        logit("No DTMF (#) before timeout", "dtmf", "")
                        dtmfrtrn += "#"
                        return dtmfrtrn
                }
        }

        //// just in case
        if dtmfrtrn == "" {
                dtmfrtrn = "#"
        }

        return dtmfrtrn
}

func oaiTranscribe(cntxt context.Context, h *ari.ChannelHandle, channame string, fname string, cname string, cid string, tenant string) string {

        logit(fname,"tmp","")

        trtrn := ""

        //// prepare the openAI client & context
        //// it seems that when multiple instances with different command line arguments, the automauic OPENAI_API_KEY from env doesn't work . . .
        oaiclient := openai.NewClient(option.WithHeader("Authorization","Bearer "+oaiKey))
        //cntxt := context.Background()

        // open the recorded file
        rfile, err := os.Open(fname)
        //rfile, err := os.Open("sound/Bulgarian.wav")
        if err != nil {
                logit("Failed to open recording file for transcription", "error", fmt.Sprintf("%v", err))
                return noAudio
        } else {
                logit(fmt.Sprintf("Opened %v successfully for transcription", fname),"info","")
        }


        //// simply transcribe
        transcription, err := oaiclient.Audio.Transcriptions.New(cntxt, openai.AudioTranscriptionNewParams{
                Model: openai.F(openai.AudioModelWhisper1),
                File:  openai.F[io.Reader](rfile),
        })
        if err != nil {
                logit("Failed to transcribe recording file", "error", fmt.Sprintf("%v", err))
                return noTranscription
        } else {
                logit("Transcribed recording file", "info","")
        }

        //// perform the transcription if required
        if((ariStyle == "translate") || (ariStyle == "transback")) { //// translate to english

                langTo := lang

                if (ariStyle == "transback") {
                        //logit(cname,"tmp",callerid)
                        //if(strings.HasPrefix(cname,"lang=")) { ////check for the lang= prefix in the imcoming CID name, as lang=serbo-croatian
                        //      langpart := strings.Split(cname,"=")[1]
                        //      reg := regexp.MustCompile("^[a-zA-Z-]+$")
                        //      if(reg.MatchString(langpart)) {
                        //              langTo = langpart
                        //      }
                        //}
                        langTo = parseLang(lang, cname)
                }

                if(langTo == "english") {
                        trfile, err := os.Open(fname)
                        if err != nil {
                                logit("Failed to open recording file for translation", "error", fmt.Sprintf("%v", err))
                                return noAudio
                        } else {
                                logit(fmt.Sprintf("Opened %v successfully for translation to %v", fname, langTo),"info","")
                        }

                        translation, err := oaiclient.Audio.Translations.New(cntxt, openai.AudioTranslationNewParams{
                                Model: openai.F(openai.AudioModelWhisper1),
                                File:  openai.F[io.Reader](trfile),
                                Prompt: openai.F(fmt.Sprintf("translate to %v",langTo)),
                                ResponseFormat: openai.F(openai.AudioResponseFormatJSON),
                                Temperature: openai.F(0.200000),
                        })
                        if err != nil {
                                logit("Failed to transcribe recording file", "error", fmt.Sprintf("%v", err))
                                return noTranscription
                        } else {
                                logit("Translated recording file", "info","")
                        }

                        if(ariStyle == "transback") {
                                trtrn = fmt.Sprintf("%v",translation.Text)
                        } else {
                                trtrn = fmt.Sprintf("%v\n\nOriginal:\n",translation.Text)
                        }

                } else {
                        trans, err := getGPTChatss(cntxt, h, channame, transcription.Text, langTo, cname, cid, tenant)
                        if err != nil {
                                logit(fmt.Sprintf("Failed to translate to %v", langTo),"error",fmt.Sprintf("%v", err))
                                } else {
                                        logit(fmt.Sprintf("Translated to %v", langTo),"info","")
                                }

                        //if err := textToSpeech(cntxt, h, trans); err != nil {
                        //      logit("TTS failed", "error", fmt.Sprintf("%v", err))
                        //}

                        xtr := "\n\nOriginal:\n"
                        if (ariStyle == "transback") { xtr = "" }

                        trtrn = fmt.Sprintf("%v%v",trans,xtr)
                }
        }

        if(ariStyle == "transback") {
                trtrn = fmt.Sprintf("%v",trtrn)
        } else {
                trtrn = fmt.Sprintf("%v%v",trtrn,transcription.Text)
        }

        return trtrn
}

func recordTranscribeReturn(cntxt context.Context, h *ari.ChannelHandle, channame string, cname string, cid string, tenant string, from string) (string, string) {

        tenantDir := fmt.Sprintf("/%v",tenant)
        if tenant == "1" {
                tenantDir = ""
        }

        recName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%v%v/%v-%v", recordingSubDir, tenantDir, strings.ReplaceAll(strings.Split(fmt.Sprintf("%v", time.Now()), ".")[0], " ", "-"), strings.Split(channame, "/")[1]),"*","s"),";","-"),":","-")

        recordIt(cntxt, h, recName, tenant, from)

        //time.Sleep(1 * time.Second)

        transcript := oaiTranscribe(cntxt, h, channame, fmt.Sprintf("%v%v.wav", recordingDir, recName), cname, cid, tenant)

        return transcript, recName

}


func textToSpeech(cntxt context.Context, h *ari.ChannelHandle, text string) error {
    logit(fmt.Sprintf("Starting text to speech with %v",rvoice), "info", "")

    tstamp := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%v", time.Now())," ",""),":",""),"-",""),".",""),"=",""),"+","")
    //logit(tstamp,"tmp","")
    tmpfilepath := fmt.Sprintf("/opt/pbxware/pw/var/lib/asterisk/sounds/%v",recordingSubDir)
    tmpfilename := fmt.Sprintf("tts-%v",tstamp)

    chrootdir := "/opt/pbxware/pw"

    // Create OpenAI client
    oaiclient := openai.NewClient(option.WithHeader("Authorization", "Bearer "+oaiKey))

    // Create temporary file to store audio
    //tmpFile, err := os.CreateTemp("/opt/pbxware/pw/var/lib/asterisk/sounds/ari", "tts-tmp.wav")
    tmpFile, err := os.Create(tmpfilepath, fmt.Sprintf("%v.wav",tmpfilename))
    if err != nil {
        logit("Failed to create temp file", "error", fmt.Sprintf("%v", err))
        return err
    }
    //defer os.Remove(tmpFile.Name()) // Clean up temp file when done

    // Get speech audio from OpenAI
    speech, err := oaiclient.Audio.Speech.New(cntxt, openai.AudioSpeechNewParams{
        Model: openai.F(openai.SpeechModelTTS1),
        Input: openai.String(text),
        //Voice: openai.F(openai.AudioSpeechNewParamsVoiceOnyx),
        Voice: openai.F(rvoice),
        ResponseFormat: openai.F(openai.AudioSpeechNewParamsResponseFormatWAV),
        Speed: openai.F(0.950000),

    })
    if err != nil {
        logit("Failed to generate speech", "error", fmt.Sprintf("%v", err))
        return err
    }
    defer speech.Body.Close()

    // Copy audio data to temp file
    _, err = io.Copy(tmpFile, speech.Body)
    if err != nil {
        logit("Failed to save audio file", "error", fmt.Sprintf("%v", err))
        return err
    }
    tmpFile.Close()

    // Prepare and play the audio file
    tmpfl := tmpFile.Name()[strings.LastIndex(tmpFile.Name(),"/")+1:]
    af := strings.Split(tmpfl,".")[0]

    //// transform the file to ulaw in PBXWare's chroot'd environment
    chrootpath := strings.Replace(tmpfilepath,chrootdir,"",1)

    soxer := exec.Command("chroot",chrootdir,"/usr/bin/sox",chrootpath+"/"+tmpfilename+".wav","--rate","8000","--channels","1","--type","ul",chrootpath+"/"+tmpfilename+".ulaw","lowpass","3400","highpass","300")
    //os.Chown(tmpfilepath+"/"+af+".ulaw",555,555)

    soxer.Run()
    //// end transform the file to ulaw in PBXWare's chroot'd environment

    //// playback the sound file
    if err := play.Play(cntxt, h, play.URI("sound:ari/"+af)).Err(); err != nil {
        logit("Failed to play TTS audio", "error", fmt.Sprintf("%v", err))
        return err
    }

    //// remove the sound files
    os.Remove(tmpfilepath+"/"+af+".wav")
    os.Remove(tmpfilepath+"/"+af+".ulaw")

    logit("Text to speech completed", "info", "")
    return nil
}


func telnyxSendGroup(msg string) { // sent as group (max 8)


        SMSToList := strings.Split(SMSTo,",")

        //// Create request payload : from & to are global variables (potentially assigned as command-line arguments)
        payload := map[string]interface{}{
                "from": SMSFrom,
                "to":   SMSToList,
                //"subject": subj,
                "text": msg,
                //"media_urls": "https://...",
        }

        //// format the payload for json
        jPayload, err := json.Marshal(payload)
        if err != nil {
                logit("Unable to format SMS payload for JSON", "error", fmt.Sprintf("%v", err))
        }

        //// prepare the request
        req, err := http.NewRequest("POST", telnyxUrlGroup, bytes.NewBuffer(jPayload))
        if err != nil {
                logit("Unable to create SMS http request", "error", fmt.Sprintf("%v", err))
        }

        //// prepare headers
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+telnyxKey)

        //// Send the request
        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
                logit("Unable to send SMS http request", "error", fmt.Sprintf("%v", err))
        }
        defer resp.Body.Close()

        //// Check http response
        if resp.StatusCode != http.StatusOK {
                logit("SMS http request failed", "error", fmt.Sprintf("%v - %v", err, resp.StatusCode))
        } else {
                logit("SMS http request successful", "info", SMSFrom+" -> "+fmt.Sprintf("%v",SMSToList))
        }

}

func telnyxSendIndiv(msg string) { // sent individually

        //// if DIDs are separated by ':', we want to send individually, not as a group.
        SMSToList := strings.Split(SMSTo,":")

        for i:= 0;i<len(SMSToList);i++ {

                //// Create request payload : from & to are global variables (potentially assigned as command-line arguments)
                payload := map[string]interface{}{
                        "from": SMSFrom,
                        "to":   SMSToList[i],
                        //"subject": subj,
                        "text": msg,
                }

                //// format the payload for json
                jPayload, err := json.Marshal(payload)
                if err != nil {
                        logit("Unable to format SMS payload for JSON", "error", fmt.Sprintf("%v", err))
                }

                //// prepare the request
                req, err := http.NewRequest("POST", telnyxUrl, bytes.NewBuffer(jPayload))
                if err != nil {
                        logit("Unable to create SMS http request", "error", fmt.Sprintf("%v", err))
                }

                //// prepare headers
                req.Header.Set("Content-Type", "application/json")
                req.Header.Set("Authorization", "Bearer "+telnyxKey)

                //// Send the request
                client := &http.Client{}
                resp, err := client.Do(req)
                if err != nil {
                        logit("Unable to send SMS http request", "error", fmt.Sprintf("%v", err))
                }
                defer resp.Body.Close()

                //// Check http response
                if resp.StatusCode != http.StatusOK {
                        logit("SMS http request failed", "error", fmt.Sprintf("%v - %v", err, resp.StatusCode))
                } else {
                        logit("SMS http request successful", "info", SMSFrom+" -> "+SMSToList[i])
                }
        }
}

func logit(inf string, typ string, errr string) {
        tnow := strings.Split(time.Now().String(), " ")
        dtstr := fmt.Sprintf("%v %v %v %v", tnow[0], tnow[1], tnow[2], tnow[3])
        logline := fmt.Sprintf("%v|%v|%v|%v|%v\n", dtstr, typ, inf, errr, ariAppName)

        //// print to stdout, if verbose is set
        if verbosity == "verbose" {
                fmt.Println(logline)
        }

        //// write to the logfile
        fl, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
                usage(myName, fmt.Sprintf("Unable to open logfile : %v", logfile))
        }
        defer fl.Close()
        if _, err := fl.WriteString(logline); err != nil {
                usage(myName, fmt.Sprintf("Unable to write logfile : %v", logfile))
        }

}

func usage(slf string, wh string) {
        fmt.Printf("\n%v\n\nUsage :\n\n\t%v IP User Pass AppName Tenant [\"SMS Name\"|-] [SMS_To|List|-] [SMS_From|-] [callbacknum|nocallbacknum|-] [en|fr|indonesian|-] [transcribe|translate|transback|chat|-] [alloy|echo|fable|onyx|nova|shimmer||-] [email_To|List|-] [LogFile|-] [DTMF_Instructions_File|-] [Recording_Instructions_File|-] [Goodbye_Sound|-] [verbose|silent|-]\n\n\t%v 127.0.0.1 ariuser aripass ariapp 999 \"Sender Name\" +15555555555 +15556667777 callbacknum - transcribe shimmer email@domain.suf /var/log/ari_log.txt dtmf_instuctions recording_instructions goodbye_sound_file silent\n\n\t%v 127.0.0.1 ariuser aripass ariapp 999 - - - - - - - - - - - - -\n\n", wh, slf, slf, slf)
}
