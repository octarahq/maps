package com.octara.maps.ui.components

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Cable
import androidx.compose.material.icons.filled.DirectionsBoat
import androidx.compose.material.icons.filled.DirectionsBus
import androidx.compose.material.icons.filled.Subway
import androidx.compose.material.icons.filled.Terrain
import androidx.compose.material.icons.filled.Train
import androidx.compose.material.icons.filled.Tram
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.PlatformTextStyle
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImagePainter
import coil.compose.rememberAsyncImagePainter
import coil.decode.SvgDecoder
import coil.request.CachePolicy
import coil.request.ImageRequest
import com.octara.maps.types.Departure
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType

fun getRoutePictoResourceUrl(pictoId: String): String {
    return "https://data.iledefrance-mobilites.fr/explore/dataset/referentiel-des-lignes/files/$pictoId/download/"
}

private fun parseHexColor(hexString: String, defaultColor: Color): Color {
    val clean = hexString.trim().removePrefix("#")
    if (clean.isEmpty()) return defaultColor
    return try {
        val colorInt = when (clean.length) {
            6 -> (0xFF000000 or clean.toLong(16)).toInt()
            8 -> clean.toLong(16).toInt()
            3 -> {
                val r = clean[0]
                val g = clean[1]
                val b = clean[2]
                (0xFF000000 or "$r$r$g$g$b$b".toLong(16)).toInt()
            }

            else -> return defaultColor
        }
        Color(colorInt)
    } catch (_: Exception) {
        defaultColor
    }
}

@Composable
fun RouteBadgeFallback(
    route: Route,
    modifier: Modifier = Modifier,
    height: Dp = 24.dp
) {
    val bgColor = parseHexColor(route.color ?: "", MaterialTheme.colorScheme.primary)
    val defaultTextColor = if (route.type == RouteType.TRAM) {
        MaterialTheme.colorScheme.onSurface
    } else {
        val luminance = 0.299 * bgColor.red + 0.587 * bgColor.green + 0.114 * bgColor.blue
        if (luminance > 0.6) Color.Black else Color.White
    }
    val textColor = if (!route.tcolor.isNullOrBlank()) {
        parseHexColor(route.tcolor, defaultTextColor)
    } else {
        defaultTextColor
    }

    val displayName = route.name.ifEmpty { (route.short_name ?: "").ifEmpty { route.id ?: "" } }

    val cornerRadius = (height * 0.22f).coerceAtLeast(3.dp)
    val fontSize = when {
        displayName.length >= 4 -> (height.value * 0.42f).sp
        displayName.length == 3 -> (height.value * 0.48f).sp
        displayName.length == 2 -> (height.value * 0.54f).sp
        else -> (height.value * 0.60f).sp
    }
    val tightTextStyle = TextStyle(
        platformStyle = PlatformTextStyle(includeFontPadding = false),
        lineHeight = fontSize
    )

    when (route.type) {
        RouteType.METRO -> {
            Box(
                modifier = modifier
                    .height(height)
                    .defaultMinSize(minWidth = height)
                    .clip(CircleShape)
                    .background(bgColor)
                    .padding(horizontal = (height * 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = displayName,
                    color = textColor,
                    fontWeight = FontWeight.Bold,
                    fontSize = fontSize,
                    style = tightTextStyle,
                    maxLines = 1,
                    softWrap = false,
                    textAlign = TextAlign.Center
                )
            }
        }

        RouteType.TRAIN -> {
            Box(
                modifier = modifier
                    .height(height)
                    .defaultMinSize(minWidth = height)
                    .clip(RoundedCornerShape(cornerRadius))
                    .background(bgColor)
                    .padding(horizontal = (height * 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = displayName,
                    color = textColor,
                    fontWeight = FontWeight.Bold,
                    fontSize = fontSize,
                    style = tightTextStyle,
                    maxLines = 1,
                    softWrap = false,
                    textAlign = TextAlign.Center
                )
            }
        }

        RouteType.TRAM -> {
            Column(
                modifier = modifier
                    .height(height)
                    .defaultMinSize(minWidth = height)
                    .wrapContentWidth(unbounded = true)
                    .padding(horizontal = 2.dp, vertical = 1.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.SpaceBetween
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(2.dp)
                        .clip(RoundedCornerShape(1.dp))
                        .background(bgColor)
                )
                Text(
                    text = displayName,
                    color = textColor,
                    fontWeight = FontWeight.Bold,
                    fontSize = fontSize,
                    style = tightTextStyle,
                    maxLines = 1,
                    softWrap = false,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(horizontal = 2.dp)
                )
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(2.dp)
                        .clip(RoundedCornerShape(1.dp))
                        .background(bgColor)
                )
            }
        }

        RouteType.BUS,
        RouteType.FERRY,
        RouteType.CABLE_TRAM,
        RouteType.AERIAL_LIFT,
        RouteType.FUNICULAR -> {
            Box(
                modifier = modifier
                    .height(height)
                    .defaultMinSize(minWidth = height)
                    .wrapContentWidth(unbounded = true)
                    .clip(RoundedCornerShape(cornerRadius))
                    .background(bgColor)
                    .padding(horizontal = (height * 0.22f)),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = displayName,
                    color = textColor,
                    fontWeight = FontWeight.Bold,
                    fontSize = fontSize,
                    style = tightTextStyle,
                    maxLines = 1,
                    softWrap = false,
                    textAlign = TextAlign.Center
                )
            }
        }
    }
}

@Composable
fun RouteIcon(
    route: Route,
    modifier: Modifier = Modifier.height(24.dp),
    height: Dp = 24.dp
) {
    val picto = route.picto ?: ""
    if (picto.isNotBlank()) {
        val pictoUrl = getRoutePictoResourceUrl(picto)
        val painter = rememberAsyncImagePainter(
            model = ImageRequest.Builder(LocalContext.current)
                .data(pictoUrl)
                .decoderFactory(SvgDecoder.Factory())
                .memoryCacheKey(pictoUrl)
                .diskCacheKey(pictoUrl)
                .memoryCachePolicy(CachePolicy.ENABLED)
                .diskCachePolicy(CachePolicy.ENABLED)
                .build()
        )
        val state = painter.state

        if (state is AsyncImagePainter.State.Error) {
            RouteBadgeFallback(route = route, modifier = modifier, height = height)
        } else {
            Image(
                painter = painter,
                contentDescription = "${route.short_name ?: ""} picto",
                modifier = modifier
                    .height(height)
                    .wrapContentWidth(),
                contentScale = ContentScale.Fit
            )
        }
    } else {
        RouteBadgeFallback(route = route, modifier = modifier, height = height)
    }
}

@Composable
fun RoutePicto(
    route: Route,
    modifier: Modifier = Modifier.height(24.dp)
) = RouteIcon(route, modifier)

@Composable
fun RouteIcon(
    route: Departure,
    modifier: Modifier = Modifier.height(24.dp)
) {
    RouteIcon(
        route = Route(
            id = route.route_id ?: "",
            short_name = route.route_name ?: "",
            type = route.type ?: RouteType.BUS,
            color = route.color ?: "",
            tcolor = "",
            picto = route.picto ?: ""
        ),
        modifier = modifier
    )
}

@Composable
fun RoutePicto(
    route: Departure,
    modifier: Modifier = Modifier.height(24.dp)
) = RouteIcon(route, modifier)

@Composable
fun RouteTypeIcon(routeType: RouteType): ImageVector {
    return when (routeType) {
        RouteType.METRO -> Icons.Filled.Subway
        RouteType.TRAIN -> Icons.Filled.Train
        RouteType.TRAM -> Icons.Filled.Tram
        RouteType.BUS -> Icons.Filled.DirectionsBus
        RouteType.FERRY -> Icons.Filled.DirectionsBoat
        RouteType.CABLE_TRAM -> Icons.Filled.Cable
        RouteType.AERIAL_LIFT -> Icons.Filled.Cable
        RouteType.FUNICULAR -> Icons.Filled.Terrain
    }
}
