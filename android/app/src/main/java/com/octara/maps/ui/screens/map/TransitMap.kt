package com.octara.maps.ui.screens.map

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.Color as AndroidColor
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.offset
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.octara.maps.types.LayerRouteLine
import com.octara.maps.types.LayerStop
import com.octara.maps.types.LegType
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.StopType
import com.octara.maps.ui.components.StopItemDisplay
import com.octara.maps.utils.api
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.maplibre.android.annotations.Icon
import org.maplibre.android.annotations.IconFactory
import org.maplibre.android.annotations.Marker
import org.maplibre.android.annotations.MarkerOptions
import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.geometry.LatLngBounds
import org.maplibre.android.maps.MapLibreMap
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style
import org.maplibre.android.style.expressions.Expression.*
import org.maplibre.android.style.layers.LineLayer
import org.maplibre.android.style.layers.Property
import org.maplibre.android.style.layers.PropertyFactory
import org.maplibre.android.style.sources.GeoJsonSource
import org.maplibre.geojson.Feature
import org.maplibre.geojson.FeatureCollection
import org.maplibre.geojson.LineString
import org.maplibre.geojson.Point
import kotlin.math.abs
import kotlin.math.asinh
import kotlin.math.pow
import kotlin.math.roundToInt
import kotlin.math.tan
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Screen

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
            color = AndroidColor.argb(70, 0, 0, 0)
        }
        canvas.drawCircle(center, center + (1.5f * density), radius, shadowPaint)

        val borderPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = AndroidColor.WHITE
            style = Paint.Style.FILL
        }
        canvas.drawCircle(center, center, radius, borderPaint)

        val fillPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = AndroidColor.parseColor("#1565C0")
            style = Paint.Style.FILL
        }
        canvas.drawCircle(center, center, radius - (2.5f * density), fillPaint)

        val dotPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
            color = AndroidColor.WHITE
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

private fun latLonToTile(lat: Double, lon: Double, zoom: Int): Pair<Int, Int> {
    val n = 2.0.pow(zoom.toDouble())
    val x = ((lon + 180.0) / 360.0 * n).toInt()
    val latRad = Math.toRadians(lat.coerceIn(-85.0511, 85.0511))
    val y = ((1.0 - asinh(tan(latRad)) / Math.PI) / 2.0 * n).toInt()
    return Pair(x, y)
}

private fun getVisibleTiles(bounds: LatLngBounds, zoom: Int, maxTiles: Int = 9): List<Pair<Int, Int>> {
    val north = bounds.latitudeNorth
    val south = bounds.latitudeSouth
    val west = bounds.longitudeWest
    val east = bounds.longitudeEast

    val (x1, y1) = latLonToTile(north, west, zoom)
    val (x2, y2) = latLonToTile(south, east, zoom)

    val minX = minOf(x1, x2)
    val maxX = maxOf(x1, x2)
    val minY = minOf(y1, y2)
    val maxY = maxOf(y1, y2)

    val tiles = mutableListOf<Pair<Int, Int>>()
    if ((maxX - minX + 1) * (maxY - minY + 1) <= maxTiles) {
        for (x in minX..maxX) {
            for (y in minY..maxY) {
                tiles.add(Pair(x, y))
            }
        }
    } else {
        val centerLat = (north + south) / 2.0
        val centerLon = (west + east) / 2.0
        val (cx, cy) = latLonToTile(centerLat, centerLon, zoom)
        for (dx in -1..1) {
            for (dy in -1..1) {
                tiles.add(Pair(cx + dx, cy + dy))
            }
        }
    }
    return tiles
}

private fun clipPointsToBounds(
    rawPts: List<List<Double>>,
    minLat: Double,
    maxLat: Double,
    minLon: Double,
    maxLon: Double
): List<List<Point>> {
    val segments = mutableListOf<List<Point>>()
    var current = mutableListOf<Point>()

    for (i in 0 until rawPts.size - 1) {
        val p1 = rawPts[i]
        val p2 = rawPts[i + 1]
        if (p1.size < 2 || p2.size < 2) continue

        val lat1 = p1[0]
        val lon1 = p1[1]
        val lat2 = p2[0]
        val lon2 = p2[1]

        val segMinLat = minOf(lat1, lat2)
        val segMaxLat = maxOf(lat1, lat2)
        val segMinLon = minOf(lon1, lon2)
        val segMaxLon = maxOf(lon1, lon2)

        val intersects = segMaxLat >= minLat && segMinLat <= maxLat &&
                segMaxLon >= minLon && segMinLon <= maxLon

        if (intersects) {
            if (current.isEmpty()) {
                current.add(Point.fromLngLat(lon1, lat1))
            }
            current.add(Point.fromLngLat(lon2, lat2))
        } else {
            if (current.size >= 2) {
                segments.add(current)
            }
            current = mutableListOf()
        }
    }
    if (current.size >= 2) {
        segments.add(current)
    }
    return segments
}

private fun stopPriority(type: StopType): Int {
    return when (type) {
        StopType.RAIL -> 5
        StopType.METRO -> 4
        StopType.STREET_TRAM -> 3
        StopType.LIFT -> 2
        StopType.STREET_BUS -> 1
        StopType.UNKNOWN -> 0
    }
}

private const val ROUTES_LINES_SOURCE_ID = "routes-lines-source"
private const val ROUTES_LINES_LAYER_ID = "routes-lines-layer"

private val routesLinesCache = java.util.concurrent.ConcurrentHashMap<String, List<LayerRouteLine>>()
private val stopsCache = java.util.concurrent.ConcurrentHashMap<String, List<LayerStop>>()

private fun initMapLayers(map: MapLibreMap) {
    val style = map.style ?: return

    if (style.getSource(ROUTES_LINES_SOURCE_ID) == null) {
        val routesSource = GeoJsonSource(ROUTES_LINES_SOURCE_ID, FeatureCollection.fromFeatures(arrayOf()))
        style.addSource(routesSource)

        val routesLayer = LineLayer(ROUTES_LINES_LAYER_ID, ROUTES_LINES_SOURCE_ID).apply {
            val dynamicWidth = interpolate(
                exponential(1.5f),
                zoom(),
                stop(8f, 1.8f),
                stop(11f, 3.0f),
                stop(14f, 5.0f),
                stop(17f, 7.5f)
            )
            setProperties(
                PropertyFactory.lineColor(get("color")),
                PropertyFactory.lineWidth(dynamicWidth),
                PropertyFactory.lineCap(Property.LINE_CAP_ROUND),
                PropertyFactory.lineJoin(Property.LINE_JOIN_ROUND)
            )
        }
        style.addLayer(routesLayer)
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
    val density = LocalDensity.current
    val router = LocalRouter.current

    var mapInstance by remember { mutableStateOf<MapLibreMap?>(null) }
    var isMapReady by remember { mutableStateOf(false) }
    var currentMarker by remember { mutableStateOf<Marker?>(null) }
    var lastAnimatedTarget by remember { mutableStateOf<LatLng?>(null) }
    var addedLayerIds by remember { mutableStateOf<List<String>>(emptyList()) }
    var addedSourceIds by remember { mutableStateOf<List<String>>(emptyList()) }

    var visibleStops by remember { mutableStateOf<List<LayerStop>>(emptyList()) }
    var cameraMoveTick by remember { mutableIntStateOf(0) }

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
                    initMapLayers(map)
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
                    } catch (_: Exception) {
                    }
                }
                addedSourceIds.forEach { id ->
                    try {
                        style.removeSource(id)
                    } catch (_: Exception) {
                    }
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
                                val source = GeoJsonSource(
                                    sourceId,
                                    FeatureCollection.fromFeature(Feature.fromGeometry(lineString))
                                )
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


    LaunchedEffect(mapInstance, isMapReady) {
        val map = mapInstance
        if (map != null && isMapReady) {
            var fetchJob: kotlinx.coroutines.Job? = null

            val cameraMoveListener = MapLibreMap.OnCameraMoveListener {
                cameraMoveTick++
            }
            map.addOnCameraMoveListener(cameraMoveListener)

            val cameraIdleListener = MapLibreMap.OnCameraIdleListener {
                cameraMoveTick++
                val camPos = map.cameraPosition
                val rawZoom = camPos.zoom

                if (rawZoom < 8.0) {
                    visibleStops = emptyList()
                    val style = map.style
                    if (style != null) {
                        (style.getSource(ROUTES_LINES_SOURCE_ID) as? GeoJsonSource)?.setGeoJson(
                            FeatureCollection.fromFeatures(arrayOf())
                        )
                    }
                    return@OnCameraIdleListener
                }

                val proj = map.projection
                val visibleRegion = try {
                    proj.visibleRegion
                } catch (_: Exception) {
                    null
                }

                val bounds = visibleRegion?.latLngBounds
                if (bounds != null) {
                    val linesZoom = rawZoom.roundToInt().coerceIn(9, 15)
                    val showStops = rawZoom >= 11.5
                    val stopsZoom = when {
                        rawZoom < 12.6 -> 10
                        rawZoom < 13.6 -> 12
                        rawZoom < 14.5 -> 14
                        rawZoom < 15.5 -> 16
                        else -> 17
                    }

                    val lineTiles = getVisibleTiles(bounds, linesZoom, maxTiles = 6)
                    val stopTiles = if (showStops) {
                        getVisibleTiles(bounds, stopsZoom, maxTiles = 12)
                    } else {
                        emptyList()
                    }

                    val minLat = bounds.latitudeSouth
                    val maxLat = bounds.latitudeNorth
                    val minLon = bounds.longitudeWest
                    val maxLon = bounds.longitudeEast

                    fetchJob?.cancel()
                    fetchJob = CoroutineScope(Dispatchers.IO).launch {
                        val collectedLines = mutableListOf<LayerRouteLine>()
                        val seenSegmentKeys = mutableSetOf<String>()

                        lineTiles.forEach { (tx, ty) ->
                            val key = "$tx/$ty/$linesZoom"
                            val lines = routesLinesCache[key] ?: try {
                                val res = api.getLayerRoutesLines(tx, ty, linesZoom)
                                routesLinesCache[key] = res.lines
                                res.lines
                            } catch (e: Exception) {
                                e.printStackTrace()
                                emptyList()
                            }

                            lines.forEach { l ->
                                val pts = l.points
                                if (pts.size >= 2) {
                                    val first = pts.first()
                                    val mid = pts[pts.size / 2]
                                    val last = pts.last()
                                    val segKey =
                                        "${l.routeId}:${pts.size}:${first.getOrNull(0)},${first.getOrNull(1)}:${
                                            mid.getOrNull(0)
                                        },${mid.getOrNull(1)}:${last.getOrNull(0)},${last.getOrNull(1)}"
                                    if (seenSegmentKeys.add(segKey)) {
                                        collectedLines.add(l)
                                    }
                                }
                            }
                        }

                        val latMargin = maxOf(0.05, (maxLat - minLat) * 0.5)
                        val lonMargin = maxOf(0.05, (maxLon - minLon) * 0.5)
                        val clipMinLat = minLat - latMargin
                        val clipMaxLat = maxLat + latMargin
                        val clipMinLon = minLon - lonMargin
                        val clipMaxLon = maxLon + lonMargin

                        val lineFeatures = mutableListOf<Feature>()
                        collectedLines.forEach { line ->
                            val rawPts = line.points
                            if (rawPts.size >= 2) {
                                val clippedSegments = clipPointsToBounds(
                                    rawPts = rawPts,
                                    minLat = clipMinLat,
                                    maxLat = clipMaxLat,
                                    minLon = clipMinLon,
                                    maxLon = clipMaxLon
                                )
                                val hexColor = if (line.color.startsWith("#")) line.color else "#${line.color}"
                                clippedSegments.forEach { segPts ->
                                    if (segPts.size >= 2) {
                                        val feat = Feature.fromGeometry(LineString.fromLngLats(segPts))
                                        feat.addStringProperty("color", hexColor)
                                        feat.addStringProperty("route_id", line.routeId)
                                        lineFeatures.add(feat)
                                    }
                                }
                            }
                        }

                        val collectedStops = mutableListOf<LayerStop>()
                        val seenStopIds = mutableSetOf<String>()
                        stopTiles.forEach { (tx, ty) ->
                            val key = "$tx/$ty/$stopsZoom"
                            val stops = stopsCache[key] ?: try {
                                val res = api.getLayerStops(tx, ty, stopsZoom)
                                stopsCache[key] = res.stops
                                res.stops
                            } catch (e: Exception) {
                                e.printStackTrace()
                                emptyList()
                            }

                            stops.forEach { st ->
                                val stopKey = "${st.id}:${st.type.value}:${st.name}"
                                if (seenStopIds.add(stopKey)) {
                                    collectedStops.add(st)
                                }
                            }
                        }

                        val stopPadLat = (maxLat - minLat) * 0.1
                        val stopPadLon = (maxLon - minLon) * 0.1
                        val inBoundsStops = collectedStops
                            .filter { st ->
                                st.lat in (minLat - stopPadLat)..(maxLat + stopPadLat) &&
                                        st.lon in (minLon - stopPadLon)..(maxLon + stopPadLon)
                            }
                            .sortedWith(
                                compareByDescending<LayerStop> { stopPriority(it.type) }
                                    .thenByDescending { it.routes.size }
                                    .thenBy { it.name }
                            )

                        withContext(Dispatchers.Main) {
                            val style = map.style
                            if (style != null) {
                                (style.getSource(ROUTES_LINES_SOURCE_ID) as? GeoJsonSource)?.setGeoJson(
                                    FeatureCollection.fromFeatures(lineFeatures)
                                )
                            }

                            if (!showStops) {
                                visibleStops = emptyList()
                                cameraMoveTick++
                                return@withContext
                            }

                            val densityVal = context.resources.displayMetrics.density
                            val (minDxDp, minDyDp, maxVisibleStops) = when {
                                rawZoom < 12.6 -> Triple(160f, 64f, 8)
                                rawZoom < 13.6 -> Triple(135f, 50f, 14)
                                rawZoom < 14.5 -> Triple(110f, 42f, 22)
                                else -> Triple(92f, 34f, 40)
                            }
                            val minDxPx = minDxDp * densityVal
                            val minDyPx = minDyDp * densityVal

                            val acceptedStops = mutableListOf<LayerStop>()
                            val acceptedScreenPts = mutableListOf<android.graphics.PointF>()

                            for (st in inBoundsStops) {
                                val screenPt = try {
                                    map.projection.toScreenLocation(LatLng(st.lat, st.lon))
                                } catch (_: Exception) {
                                    null
                                } ?: continue

                                val overlaps = acceptedScreenPts.any { existing ->
                                    abs(screenPt.x - existing.x) < minDxPx &&
                                            abs(screenPt.y - existing.y) < minDyPx
                                }
                                if (!overlaps) {
                                    acceptedStops.add(st)
                                    acceptedScreenPts.add(screenPt)
                                    if (acceptedStops.size >= maxVisibleStops) break
                                }
                            }

                            visibleStops = acceptedStops
                            cameraMoveTick++
                        }
                    }
                }
            }

            map.addOnCameraIdleListener(cameraIdleListener)
            cameraIdleListener.onCameraIdle()
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

    val iconHalfPx = with(density) { 10.dp.roundToPx() }

    Box(modifier = modifier) {
        AndroidView(
            factory = { mapView },
            modifier = Modifier.fillMaxSize()
        )

        val map = mapInstance
        if (map != null && isMapReady && activeJourney == null) {
            visibleStops.forEach { stop ->
                key("${stop.id}:${stop.type.value}:${stop.name}:${stop.lat}:${stop.lon}") {
                    StopItemDisplay(
                        stop = stop,
                        withBG = true,
                        onClick = {
                            router.navigateTo(
                                Screen.StopInfo(
                                    stopId = stop.id,
                                    lat = stop.lat.toDouble(),
                                    lon = stop.lon.toDouble(),
                                    name = stop.name
                                ),
                                popCurrent = true
                            )
                        },
                        modifier = Modifier.offset {
                            @Suppress("UNUSED_EXPRESSION")
                            cameraMoveTick
                            val screenPt = try {
                                map.projection.toScreenLocation(LatLng(stop.lat, stop.lon))
                            } catch (_: Exception) {
                                null
                            }
                            if (screenPt != null) {
                                IntOffset(
                                    x = screenPt.x.roundToInt() - iconHalfPx,
                                    y = screenPt.y.roundToInt() - iconHalfPx
                                )
                            } else {
                                IntOffset(-10000, -10000)
                            }
                        }
                    )
                }
            }
        }
    }
}
