package codec

import (
	/*
		#cgo CFLAGS: -I/usr/local/include
		#cgo LDFLAGS: -L/usr/local/lib  -lavformat -lavcodec -lavresample -lavutil -lx264 -lz -ldl -lm

		#include "libavcodec/avcodec.h"
		#include "libavutil/avutil.h"
		#include "libavformat/avformat.h"

		typedef struct {
			AVCodec *c;
			AVCodecContext *ctx;
			AVFrame *f;
			int got;
		} h264dec_t ;

		static int h264dec_new(h264dec_t *h, uint8_t *data, int len) {
			h->c = avcodec_find_decoder(AV_CODEC_ID_H264);

			h->ctx = avcodec_alloc_context3(h->c);
			h->ctx->extradata = data;
			h->ctx->extradata_size = len;
			h->f = av_frame_alloc();

			return avcodec_open2(h->ctx, h->c, 0);
		}

		static int h264dec_release(h264dec_t *m) {
			// release context
			avcodec_close(m->ctx);
			av_free(m->ctx);

			// release frame
			av_frame_free(&m->f);
		}

		static int h264dec_decode(h264dec_t *h, uint8_t *data, int len) {
			AVPacket pkt;
			av_init_packet(&pkt);

			if (data != NULL) {
				pkt.data = data;
				pkt.size = len;
			}

			int r = avcodec_decode_video2(h->ctx, h->f, &h->got, &pkt);

			static char error_buffer[255];
			if (r < 0) {
				av_strerror(r, error_buffer, sizeof(error_buffer));
				av_log(h->ctx, AV_LOG_DEBUG, "Video decode error: %s\n", error_buffer);
			}

			return r;
		}

		static int h264dec_decode2(h264dec_t *h, uint8_t *data, int len, int64_t pts, int64_t dts, AVFrame *frame) {
			AVPacket pkt;
			av_init_packet(&pkt);

			pkt.data = data;
			pkt.size = len;
			pkt.pts  = pts;
			pkt.dts  = dts;

			int r = avcodec_decode_video2(h->ctx, frame, &h->got, &pkt);

			static char error_buffer[255];
			if (r < 0) {
				av_strerror(r, error_buffer, sizeof(error_buffer));
				av_log(h->ctx, AV_LOG_DEBUG, "Video decode error: %s\n", error_buffer);
			}

			return r;
		}
	*/
	"C"
	"errors"
	"image"
	"unsafe"
)
import "log"

type H264Decoder struct {
	m C.h264dec_t
}

func NewH264Decoder(header []byte) (m *H264Decoder, err error) {
	m = &H264Decoder{}

	avLock.Lock()
	defer avLock.Unlock()

	r := C.h264dec_new(
		&m.m,
		(*C.uint8_t)(unsafe.Pointer(&header[0])),
		(C.int)(len(header)),
	)

	if int(r) < 0 {
		err = errors.New("open codec failed")
	}

	return
}

func (m *H264Decoder) Release() {
	log.Printf("Decoder released")

	C.h264dec_release(&m.m)
}

func (m *H264Decoder) Decode(nal []byte) (f *image.YCbCr, err error) {
	r := C.h264dec_decode(
		&m.m,
		(*C.uint8_t)(unsafe.Pointer(&nal[0])),
		(C.int)(len(nal)),
	)
	if int(r) < 0 {
		err = errors.New("decode failed")
		return
	}
	if m.m.got == 0 {
		err = errors.New("no picture")
		return
	}

	w := int(m.m.f.width)
	h := int(m.m.f.height)
	ys := int(m.m.f.linesize[0])
	cs := int(m.m.f.linesize[1])

	f = &image.YCbCr{
		Y:              fromCPtr(unsafe.Pointer(m.m.f.data[0]), ys*h),
		Cb:             fromCPtr(unsafe.Pointer(m.m.f.data[1]), cs*h/2),
		Cr:             fromCPtr(unsafe.Pointer(m.m.f.data[2]), cs*h/2),
		YStride:        ys,
		CStride:        cs,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, w, h),
	}

	return
}

func (m *H264Decoder) Decode2(avPacket *AVPacket, avFrame *AVFrame) (got bool, err error) {
	p := (*C.uint8_t)(unsafe.Pointer(nil))
	l := (C.int)(0)
	if avPacket.Data != nil {
		p = (*C.uint8_t)(unsafe.Pointer(&avPacket.Data[0]))
		l = (C.int)(len(avPacket.Data))
	}

	r := C.h264dec_decode2(
		&m.m,
		p,
		l,
		(C.int64_t)(avPacket.Pts),
		(C.int64_t)(avPacket.Dts),
		avFrame.f,
	)

	if int(r) < 0 {
		err = errors.New("decode failed")

		return
	}

	got = m.m.got > 0

	return
}
