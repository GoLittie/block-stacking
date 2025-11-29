// @ts-ignore
import express from "express"
import * as path from "node:path";
const app = express()

app.get("/", (req, res) => {
    res.sendFile(path.resolve(__dirname, "../game/game.html"))
})

app.get("/wasm_exec.js", (req, res) => {
    res.sendFile(path.resolve(__dirname, "../game/wasm_exec.js"))
})

app.get("/main.wasm", (req, res) => {
    res.sendFile(path.resolve(__dirname, "../game/main.wasm"))
})

app.listen(3000)