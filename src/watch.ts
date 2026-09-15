export function buildWatchUrl(rtspBaseUrl: string, pathName: string): string {
  const encodedPath = pathName
    .split("/")
    .map((segment) => encodeURIComponent(segment))
    .join("/");

  return `${rtspBaseUrl}/${encodedPath}`;
}

export function watchStream(
  pathName: string,
  rtspBaseUrl: string,
  player: string,
): string {
  const url = buildWatchUrl(rtspBaseUrl, pathName);

  Bun.spawn(
    [
      player,
      "-autoexit",
      "-window_title",
      pathName,
      "-rtsp_transport",
      "tcp",
      "-loglevel",
      "quiet",
      url,
    ],
    {
      stdout: "ignore",
      stderr: "ignore",
      stdin: "ignore",
      detached: true,
    },
  );

  return url;
}
