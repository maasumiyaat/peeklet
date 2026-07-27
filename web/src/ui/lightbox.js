import BiggerPicture from "bigger-picture";
import "bigger-picture/css";

let bp;

function ensure() {
  if (!bp) bp = BiggerPicture({ target: document.body });
  return bp;
}

function videoMime(name) {
  const ext = name.split(".").pop().toLowerCase();
  return (
    {
      mp4: "video/mp4",
      webm: "video/webm",
      mov: "video/quicktime",
      m4v: "video/mp4",
      ogg: "video/ogg",
    }[ext] || "video/mp4"
  );
}

// Map our API's file objects -> Bigger Picture items.
function toItem(f) {
  if (f.type === "video") {
    return {
      sources: JSON.stringify([{ src: f.url, type: videoMime(f.name) }]),
      alt: f.name,
      caption: f.name,
    };
  }
  return {
    img: f.url,
    thumb: f.url,
    alt: f.name,
    caption: f.name,
  };
}

// Open the lightbox on `files` (array of {name,url,type}) at index `position`.
export function openLightbox(files, position) {
  ensure().open({
    items: files.map(toItem),
    position,
    intro: "fadeup",
  });
}