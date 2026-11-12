use serde_json::{json, Value};
use std::io::{BufRead, BufReader, Write};
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};

pub struct Client {
    child: Child,
    stdin: ChildStdin,
    stdout: BufReader<ChildStdout>,
    next: u64,
}

impl Client {
    pub fn start(binary: &str, args: &[&str]) -> std::io::Result<Self> {
        let mut child = Command::new(binary)
            .args(args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()?;
        let stdin = child.stdin.take().expect("stdin");
        let stdout = BufReader::new(child.stdout.take().expect("stdout"));
        Ok(Self {
            child,
            stdin,
            stdout,
            next: 0,
        })
    }

    pub fn call(&mut self, method: &str, params: Value) -> Result<Value, String> {
        self.next += 1;
        let id = self.next;
        let request = json!({ "id": id, "method": method, "params": params });
        writeln!(self.stdin, "{request}").map_err(|error| error.to_string())?;
        self.stdin.flush().map_err(|error| error.to_string())?;

        let mut line = String::new();
        self.stdout
            .read_line(&mut line)
            .map_err(|error| error.to_string())?;
        let response: Value = serde_json::from_str(&line).map_err(|error| error.to_string())?;
        if let Some(error) = response.get("error") {
            let code = error
                .get("code")
                .and_then(Value::as_str)
                .unwrap_or("error");
            let message = error
                .get("message")
                .and_then(Value::as_str)
                .unwrap_or("unknown");
            return Err(format!("{code}: {message}"));
        }
        Ok(response.get("result").cloned().unwrap_or(Value::Null))
    }

    pub fn version(&mut self) -> Result<String, String> {
        let result = self.call("version", json!({}))?;
        result
            .get("version")
            .and_then(Value::as_str)
            .map(str::to_owned)
            .ok_or_else(|| "missing version".to_owned())
    }

    pub fn parse(&mut self, body: &str, format: &str) -> Result<Value, String> {
        self.call("parse", json!({ "body": body, "format": format }))
    }

    pub fn compile(
        &mut self,
        body: &str,
        format: &str,
        target: &str,
        port: u16,
        strict: bool,
    ) -> Result<Value, String> {
        self.call(
            "compile",
            json!({
                "body": body,
                "format": format,
                "target": target,
                "port": port,
                "strict": strict
            }),
        )
    }

    pub fn run(
        &mut self,
        body: &str,
        format: &str,
        target: &str,
        binary: &str,
        port: u16,
        strict: bool,
    ) -> Result<Value, String> {
        self.call(
            "run",
            json!({
                "body": body,
                "format": format,
                "target": target,
                "binary": binary,
                "port": port,
                "strict": strict
            }),
        )
    }

    pub fn stop(&mut self, session_id: &str) -> Result<Value, String> {
        self.call("stop", json!({ "session_id": session_id }))
    }

    pub fn close(mut self) -> std::io::Result<()> {
        drop(self.stdin);
        self.child.wait().map(|_| ())
    }
}
