package com.octara.maps.ui.screens.map

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import org.maplibre.android.annotations.Icon
import org.maplibre.android.annotations.IconFactory
import org.maplibre.android.annotations.Marker
import org.maplibre.android.annotations.MarkerOptions
import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.MapLibreMap
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style

import android.graphics.Color as AndroidColor
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.Property
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point
import com.octara.maps.types.LegType
import com.octara.maps.types.ParsedTrip
import org.maplibre.android.style.expressions.Expression.*

private const val OSM_STYLE_JSON = """
{
  "version": 8,
  "sources": {
    "osm-raster-tiles": {
      "type": "raster",
      "tiles": [
        "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
      ],
      "tileSize": 256
    }
  },
  "layers": [
    {
      "id": "osm-tiles",
      "type": "raster",
      "source": "osm-raster-tiles",
      "minzoom": 0,
      "maxzoom": 19
    }
  ]
}
"""

fun createTransitMarkerIcon(context: Context): Icon? {
    return try {
        val density = context.resources.displayMetrics.density
        val sizePx = (36 * density).toInt().coerceAtLeast(1)
        val bitmap = Bitmap.createBitmap(sizePx, sizePx, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)

        val center = sizePx / 2f
        val radius = (sizePx / 2f) - (2f * density)

        val shadowPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = android.graphics.Color.argb(70, 0, 0, 0)
        }
        canvas.drawCircle(center, center + (1.5f * density), radius, shadowPaint)

        val borderPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = android.graphics.Color.WHITE
            style = Paint.Style.FILL
        }
        canvas.drawCircle(center, center, radius, borderPaint)

        val fillPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = android.graphics.Color.parseColor("#1565C0")
            style = Paint.Style.FILL
        }
        canvas.drawCircle(center, center, radius - (2.5f * density), fillPaint)

        val dotPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = android.graphics.Color.WHITE
            style = Paint.Style.FILL
        }
        canvas.drawCircle(center, center, 4.5f * density, dotPaint)

        IconFactory.getInstance(context).fromBitmap(bitmap)
    } catch (e: Exception) {
        try {
            IconFactory.getInstance(context).defaultMarker()
        } catch (_: Exception) {
            null
        }
    }
}

@Composable
fun TransitMap(
    modifier: Modifier = Modifier,
    initialCenter: LatLng = LatLng(48.8566, 2.3522),
    initialZoom: Double = 11.0,
    cameraTarget: LatLng? = null,
    cameraZoom: Double = 15.5,
    bottomPaddingPx: Int = 0,
    markerPosition: LatLng? = null,
    markerTitle: String? = null,
    activeJourney: ParsedTrip? = null,
    animationDurationMs: Int = 1200,
    onMapReady: ((MapLibreMap) -> Unit)? = null
) {
    val context = LocalContext.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle

    var mapInstance by remember { mutableStateOf<MapLibreMap?>(null) }
    var isMapReady by remember { mutableStateOf(false) }
    var currentMarker by remember { mutableStateOf<Marker?>(null) }
    var lastAnimatedTarget by remember { mutableStateOf<LatLng?>(null) }
    var addedLayerIds by remember { mutableStateOf<List<String>>(emptyList()) }
    var addedSourceIds by remember { mutableStateOf<List<String>>(emptyList()) }

    val mapView = remember {
        MapView(context).apply {
            onCreate(null)
            getMapAsync { map ->
                map.uiSettings.isLogoEnabled = false
                map.uiSettings.isAttributionEnabled = false

                map.setStyle(Style.Builder().fromJson(OSM_STYLE_JSON)) {
                    val startCenter = cameraTarget ?: initialCenter
                    val startZoom = if (cameraTarget != null) cameraZoom else initialZoom
                    if (bottomPaddingPx > 0) {
                        map.setPadding(0, 0, 0, bottomPaddingPx)
                    }
                    map.cameraPosition = CameraPosition.Builder()
                        .target(startCenter)
                        .zoom(startZoom)
                        .build()

                    mapInstance = map
                    isMapReady = true
                    onMapReady?.invoke(map)
                }
            }
        }
    }

    LaunchedEffect(cameraTarget, cameraZoom, bottomPaddingPx, mapInstance, isMapReady) {
        val map = mapInstance
        if (map != null && isMapReady) {
            map.setPadding(0, 0, 0, bottomPaddingPx)

            if (cameraTarget != null && cameraTarget != lastAnimatedTarget) {
                lastAnimatedTarget = cameraTarget
                val update = CameraUpdateFactory.newLatLngZoom(cameraTarget, cameraZoom)
                map.animateCamera(update, animationDurationMs)
            } else if (cameraTarget == null) {
                lastAnimatedTarget = null
            }
        }
    }

    LaunchedEffect(markerPosition, markerTitle, mapInstance, isMapReady) {
        val map = mapInstance
        if (map != null && isMapReady) {
            currentMarker?.let {
                try {
                    map.removeMarker(it)
                } catch (_: Exception) {
                    
                }
                currentMarker = null
            }

            if (markerPosition != null) {
                val markerOptions = MarkerOptions()
                    .position(markerPosition)
                markerTitle?.let { markerOptions.title(it) }

                val icon = createTransitMarkerIcon(context)
                if (icon != null) {
                    markerOptions.icon(icon)
                }
                currentMarker = map.addMarker(markerOptions)
            }
        }
    }

    LaunchedEffect(activeJourney, mapInstance, isMapReady) {
        val map = mapInstance
        if (map != null && isMapReady) {
            val style = map.style
            if (style != null) {
                addedLayerIds.forEach { id ->
                    try {
                        style.removeLayer(id)
                    } catch (_: Exception) {}
                }
                addedSourceIds.forEach { id ->
                    try {
                        style.removeSource(id)
                    } catch (_: Exception) {}
                }

                val newLayerIds = mutableListOf<String>()
                val newSourceIds = mutableListOf<String>()
                val allPoints = mutableListOf<LatLng>()

                activeJourney?.legs?.forEachIndexed { index, leg ->
                    val coords = leg.coordinates
                    if (!coords.isNullOrEmpty()) {
                        val points = mutableListOf<Point>()
                        for (coord in coords) {
                            if (coord.size >= 2) {
                                val lat = coord[0]
                                val lon = coord[1]
                                points.add(Point.fromLngLat(lon, lat))
                                allPoints.add(LatLng(lat, lon))
                            }
                        }

                        if (points.size >= 2) {
                            val lineString = LineString.fromLngLats(points)
                            val sourceId = "journey-source-$index"
                            val layerId = "journey-layer-$index"

                            try {
                                val source = GeoJsonSource(sourceId, FeatureCollection.fromFeature(Feature.fromGeometry(lineString)))
                                style.addSource(source)
                                newSourceIds.add(sourceId)

                                val isDashed = leg.type == LegType.WALK || leg.type == LegType.BIKE
                                val hexColor = if (isDashed) {
                                    "#757575"
                                } else {
                                    val c = leg.route?.color
                                    if (c != null && c.isNotEmpty()) {
                                        if (c.startsWith("#")) c else "#$c"
                                    } else {
                                        "#0071BD"
                                    }
                                }

                                val parsedColor = try {
                                    AndroidColor.parseColor(hexColor)
                                } catch (_: Exception) {
                                    if (isDashed) AndroidColor.GRAY else AndroidColor.parseColor("#0071BD")
                                }

                                val layer = LineLayer(layerId, sourceId).apply {
                                    val dynamicLineWidth = if (isDashed) {
                                        interpolate(
                                            exponential(1.5f),
                                            zoom(),
                                            stop(10f, 1.8f),
                                            stop(13f, 3.0f),
                                            stop(16f, 4.5f)
                                        )
                                    } else {
                                        interpolate(
                                            exponential(1.5f),
                                            zoom(),
                                            stop(9f, 2.5f),
                                            stop(12f, 4.5f),
                                            stop(15f, 7.5f),
                                            stop(18f, 10.0f)
                                        )
                                    }

                                    setProperties(
                                        PropertyFactory.lineColor(parsedColor),
                                        PropertyFactory.lineWidth(dynamicLineWidth),
                                        PropertyFactory.lineCap(Property.LINE_CAP_ROUND),
                                        PropertyFactory.lineJoin(Property.LINE_JOIN_ROUND)
                                    )
                                    if (isDashed) {
                                        setProperties(
                                            PropertyFactory.lineDasharray(arrayOf(1.8f, 1.8f))
                                        )
                                    }
                                }
                                style.addLayer(layer)
                                newLayerIds.add(layerId)
                            } catch (e: Exception) {
                                e.printStackTrace()
                            }
                        }
                    }
                }

                addedLayerIds = newLayerIds
                addedSourceIds = newSourceIds

                if (allPoints.isNotEmpty()) {
                    try {
                        val boundsBuilder = LatLngBounds.Builder()
                        allPoints.forEach { boundsBuilder.include(it) }
                        val bounds = boundsBuilder.build()
                        val paddingPx = (64 * context.resources.displayMetrics.density).toInt()
                        val update = CameraUpdateFactory.newLatLngBounds(bounds, paddingPx)
                        map.animateCamera(update, 1000)
                    } catch (e: Exception) {
                        e.printStackTrace()
                    }
                }
            }
        }
    }

    DisposableEffect(lifecycle, mapView) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> mapView.onStart()
                Lifecycle.Event.ON_RESUME -> mapView.onResume()
                Lifecycle.Event.ON_PAUSE -> mapView.onPause()
                Lifecycle.Event.ON_STOP -> mapView.onStop()
                Lifecycle.Event.ON_DESTROY -> mapView.onDestroy()
                else -> {}
            }
        }
        lifecycle.addObserver(observer)
        onDispose {
            lifecycle.removeObserver(observer)
        }
    }

    AndroidView(
        factory = { mapView },
        modifier = modifier
    )
}
