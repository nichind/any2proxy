use any2proxy::Client;

#[test]
fn version_parse_compile() {
    let mut client = Client::start("go", &["run", "../../cmd/a2pd"]).unwrap();
    assert!(!client.version().unwrap().is_empty());

    let profile = client
        .parse("vless://u@example.com:443#one", "plain")
        .unwrap();
    assert_eq!(profile["nodes"][0]["tag"], "one");

    let compiled = client
        .compile("vless://u@example.com:443#one", "plain", "mihomo", 1080, false)
        .unwrap();
    assert!(compiled["config"].as_str().unwrap().contains("socks-port"));

    client.close().unwrap();
}
