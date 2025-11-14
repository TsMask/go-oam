package service

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "path/filepath"

    "github.com/tsmask/go-oam/framework/utils/file"
    "github.com/tsmask/go-oam/framework/ws"
    "github.com/tsmask/go-oam/modules/ws/model"
    "github.com/tsmask/go-oam/modules/ws/processor"
)

// Commont 通用
func ReceiveCommont(conn *ws.ServerConn, msg []byte) {
    var reqMsg model.WSRequest
    if err := json.Unmarshal(msg, &reqMsg); err != nil {
        SendErr(conn, "", "message format json error")
        return
	}

	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		SendErr(conn, "", "message requestId is required")
		return
	}

    switch reqMsg.Type {
    case "close":
        conn.Close()
        return
    case "ping", "PING":
        conn.Pong()
        SendOK(conn, reqMsg.RequestID, "PONG")
        return
    case "ps":
        data, err := processor.GetProcessData(reqMsg.Data)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, err.Error())
            return
        }
        SendOK(conn, reqMsg.RequestID, data)
    case "net":
        data, err := processor.GetNetConnections(reqMsg.Data)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, err.Error())
            return
        }
        SendOK(conn, reqMsg.RequestID, data)
    case "file-upload":
        bodyBytes, err := json.Marshal(reqMsg.Data)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, "message data format error")
            return
        }
        var body struct {
            FileName   string `json:"fileName"`
            Identifier string `json:"identifier"`
            Index      string `json:"index"`
            Content    string `json:"content"`
        }
        if err := json.Unmarshal(bodyBytes, &body); err != nil {
            SendErr(conn, reqMsg.RequestID, "message data format error")
            return
        }
        if body.FileName == "" || body.Content == "" {
            SendErr(conn, reqMsg.RequestID, "fileName and content must be set")
            return
        }
        bin, err := base64.StdEncoding.DecodeString(body.Content)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, "content base64 decode error")
            return
        }
        if body.Identifier != "" && body.Index != "" {
            p, err := file.TransferChunkUploadBytes(body.FileName, body.Index, body.Identifier, bin)
            if err != nil {
                SendErr(conn, reqMsg.RequestID, err.Error())
                return
            }
            SendOK(conn, reqMsg.RequestID, p)
            return
        }
        p, err := file.TransferUploadBytes(body.FileName, bin, []string{})
        if err != nil {
            SendErr(conn, reqMsg.RequestID, err.Error())
            return
        }
        SendOK(conn, reqMsg.RequestID, map[string]string{
            "filePath":         p,
            "newFileName":      filepath.Base(p),
            "originalFileName": body.FileName,
        })
    case "file-download":
        bodyBytes, err := json.Marshal(reqMsg.Data)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, "message data format error")
            return
        }
        var body struct {
            FilePath string `json:"filePath"`
            Range    string `json:"range"`
        }
        if err := json.Unmarshal(bodyBytes, &body); err != nil {
            SendErr(conn, reqMsg.RequestID, "message data format error")
            return
        }
        if body.FilePath == "" {
            SendErr(conn, reqMsg.RequestID, "filePath must be set")
            return
        }
        res, err := file.ReadUploadFileStream(body.FilePath, body.Range)
        if err != nil {
            SendErr(conn, reqMsg.RequestID, err.Error())
            return
        }
        dataBytes, _ := json.Marshal(res["data"])
        var raw []byte
        if b, ok := res["data"].([]byte); ok {
            raw = b
        } else {
            _ = json.Unmarshal(dataBytes, &raw)
        }
        SendOK(conn, reqMsg.RequestID, map[string]any{
            "range":     res["range"],
            "chunkSize": res["chunkSize"],
            "fileSize":  res["fileSize"],
            "data":      base64.StdEncoding.EncodeToString(raw),
        })
    default:
        SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
        return
    }
}
