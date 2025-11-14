package service

import (
    "bytes"
    "encoding/json"
    "fmt"
    "path/filepath"

    "github.com/tsmask/go-oam/framework/utils/file"
    "github.com/tsmask/go-oam/framework/ws"
)

type binHeader struct {
    RequestID string `json:"requestId"`
    Op        string `json:"op"`
    Msg       string `json:"msg,omitempty"`
    Code      int    `json:"code"`
    FilePath  string `json:"filePath,omitempty"`
    NewName   string `json:"newFileName,omitempty"`
    OrigName  string `json:"originalFileName,omitempty"`
    Range     string `json:"range,omitempty"`
    ChunkSize int64  `json:"chunkSize,omitempty"`
    FileSize  int64  `json:"fileSize,omitempty"`
}

func packBinary(h binHeader, payload []byte) []byte {
    hb, _ := json.Marshal(h)
    return append(append(hb, []byte("\n\n")...), payload...)
}

func ReceiveBinary(conn *ws.ServerConn, msg []byte) {
    parts := bytes.SplitN(msg, []byte("\n\n"), 2)
    if len(parts) < 1 {
        conn.Send(packBinary(binHeader{RequestID: "", Op: "", Msg: "binary header missing", Code: 1}, nil))
        return
    }
    var header struct {
        RequestID string `json:"requestId"`
        Op        string `json:"op"`
        FileName  string `json:"fileName"`
        Identifier string `json:"identifier"`
        Index     string `json:"index"`
        FilePath  string `json:"filePath"`
        Range     string `json:"range"`
    }
    if err := json.Unmarshal(parts[0], &header); err != nil {
        conn.Send(packBinary(binHeader{RequestID: "", Op: "", Msg: "binary header json error", Code: 1}, nil))
        return
    }
    if header.RequestID == "" {
        conn.Send(packBinary(binHeader{RequestID: "", Op: header.Op, Msg: "requestId is required", Code: 1}, nil))
        return
    }
    switch header.Op {
    case "ping":
        conn.Pong()
        conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: "pong", Code: 0}, nil))
        return
    case "upload":
        if len(parts) != 2 || header.FileName == "" {
            conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: "fileName or payload missing", Code: 1}, nil))
            return
        }
        if header.Identifier != "" && header.Index != "" {
            p, err := file.TransferChunkUploadBytes(header.FileName, header.Index, header.Identifier, parts[1])
            if err != nil {
                conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: err.Error(), Code: 1}, nil))
                return
            }
            conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Code: 0, FilePath: p}, nil))
            return
        }
        p, err := file.TransferUploadBytes(header.FileName, parts[1], []string{})
        if err != nil {
            conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: err.Error(), Code: 1}, nil))
            return
        }
        conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Code: 0, FilePath: p, NewName: filepath.Base(p), OrigName: header.FileName}, nil))
    case "download":
        if header.FilePath == "" {
            conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: "filePath missing", Code: 1}, nil))
            return
        }
        res, err := file.ReadUploadFileStream(header.FilePath, header.Range)
        if err != nil {
            conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: err.Error(), Code: 1}, nil))
            return
        }
        var data []byte
        if b, ok := res["data"].([]byte); ok {
            data = b
        }
        h := binHeader{RequestID: header.RequestID, Op: header.Op, Code: 0}
        if v, ok := res["range"].(string); ok { h.Range = v }
        if v, ok := res["chunkSize"].(int64); ok { h.ChunkSize = v }
        if v, ok := res["fileSize"].(int64); ok { h.FileSize = v }
        conn.Send(packBinary(h, data))
    default:
        conn.Send(packBinary(binHeader{RequestID: header.RequestID, Op: header.Op, Msg: fmt.Sprintf("op %s not supported", header.Op), Code: 1}, nil))
        return
    }
}

